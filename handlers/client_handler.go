package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"panel/database"
)

type XrayManager interface {
	Reload() error
}

type TrafficResetter interface {
	ResetClient(id int64)
}

type ClientHandler struct {
	TunnelManager interface {
		VLESSURL() string
		TrojanURL() string
	}

	XrayManager    XrayManager
	TrafficResetter TrafficResetter
}

func NewClientHandler(
	tunnelManager interface {
		VLESSURL() string
		TrojanURL() string
	},
	xrayManager XrayManager,
	trafficResetter TrafficResetter,
) *ClientHandler {

	return &ClientHandler{
		TunnelManager:   tunnelManager,
		XrayManager:     xrayManager,
		TrafficResetter: trafficResetter,
	}
}

// Create creates a new client.
func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	protocol := strings.TrimSpace(r.FormValue("protocol"))
	quotaValue := strings.TrimSpace(r.FormValue("quota"))

	if name == "" {
		http.Error(w, "client name is required", http.StatusBadRequest)
		return
	}

	if protocol != "vless" && protocol != "trojan" {
		http.Error(w, "invalid protocol", http.StatusBadRequest)
		return
	}

	trafficLimit, err := parseQuotaGB(quotaValue)
	if err != nil {
		http.Error(w, "invalid quota", http.StatusBadRequest)
		return
	}

	var clientUUID string
	var password string

	if protocol == "vless" {
		clientUUID = uuid.New().String()
	} else {
		password = generatePassword(24)
	}

	_, err = database.DB.Exec(`
		INSERT INTO clients
		(
			name,
			protocol,
			uuid,
			password,
			created_at,
			traffic_limit_bytes,
			traffic_used_bytes,
			enabled,
			last_seen
		)
		VALUES (?, ?, ?, ?, ?, ?, 0, 1, NULL)
	`,
		name,
		protocol,
		clientUUID,
		password,
		time.Now(),
		trafficLimit,
	)

	if err != nil {
		http.Error(w, "failed to create client", http.StatusInternalServerError)
		return
	}

	if h.XrayManager != nil {
		if err := h.XrayManager.Reload(); err != nil {
			http.Error(
				w,
				"client was saved but Xray reload failed: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Delete deletes a client.
func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid client id", http.StatusBadRequest)
		return
	}

	_, err = database.DB.Exec(
		"DELETE FROM clients WHERE id = ?",
		id,
	)

	if err != nil {
		http.Error(w, "failed to delete client", http.StatusInternalServerError)
		return
	}

	if h.XrayManager != nil {
		if err := h.XrayManager.Reload(); err != nil {
			http.Error(
				w,
				"client was deleted but Xray reload failed: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// EditQuota changes the traffic quota of a client.
func (h *ClientHandler) EditQuota(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid client id", http.StatusBadRequest)
		return
	}

	quotaValue := strings.TrimSpace(r.FormValue("quota"))

	trafficLimit, err := parseQuotaGB(quotaValue)
	if err != nil {
		http.Error(w, "invalid quota", http.StatusBadRequest)
		return
	}

	var trafficUsed int64

	err = database.DB.QueryRow(`
		SELECT traffic_used_bytes
		FROM clients
		WHERE id = ?
	`, id).Scan(&trafficUsed)

	if err != nil {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	enabled := 1

	if trafficLimit > 0 && trafficUsed >= trafficLimit {
		enabled = 0
	}

	_, err = database.DB.Exec(`
		UPDATE clients
		SET traffic_limit_bytes = ?,
		    enabled = ?
		WHERE id = ?
	`,
		trafficLimit,
		enabled,
		id,
	)

	if err != nil {
		http.Error(w, "failed to update quota", http.StatusInternalServerError)
		return
	}

	if h.XrayManager != nil {
		if err := h.XrayManager.Reload(); err != nil {
			http.Error(
				w,
				"quota was updated but Xray reload failed: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ResetTraffic resets the accumulated traffic of a client.
func (h *ClientHandler) ResetTraffic(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid client id", http.StatusBadRequest)
		return
	}

	_, err = database.DB.Exec(`
		UPDATE clients
		SET traffic_used_bytes = 0,
		    last_seen = NULL,
		    enabled = 1
		WHERE id = ?
	`, id)

	if err != nil {
		http.Error(w, "failed to reset traffic", http.StatusInternalServerError)
		return
	}

	// Tell the collector that the old Xray counter
	// must become the new baseline.
	if h.TrafficResetter != nil {
		h.TrafficResetter.ResetClient(id)
	}

	if h.XrayManager != nil {
		if err := h.XrayManager.Reload(); err != nil {
			http.Error(
				w,
				"traffic was reset but Xray reload failed: "+err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Config generates the VLESS/Trojan configuration.
func (h *ClientHandler) Config(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid client id", http.StatusBadRequest)
		return
	}

	var (
		name       string
		protocol   string
		clientUUID string
		password   string
		enabled    int
	)

	err = database.DB.QueryRow(`
		SELECT
			name,
			protocol,
			uuid,
			password,
			enabled
		FROM clients
		WHERE id = ?
	`, id).Scan(
		&name,
		&protocol,
		&clientUUID,
		&password,
		&enabled,
	)

	if err != nil {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	if enabled == 0 {
		http.Error(w, "client is disabled", http.StatusForbidden)
		return
	}

	if h.TunnelManager == nil {
		http.Error(
			w,
			"tunnel manager unavailable",
			http.StatusServiceUnavailable,
		)
		return
	}

	var tunnelURL string

	if protocol == "vless" {
		tunnelURL = h.TunnelManager.VLESSURL()
	} else {
		tunnelURL = h.TunnelManager.TrojanURL()
	}

	if tunnelURL == "" {
		http.Error(
			w,
			"tunnel is not ready yet",
			http.StatusServiceUnavailable,
		)
		return
	}

	parsedURL, err := url.Parse(tunnelURL)
	if err != nil {
		http.Error(
			w,
			"invalid tunnel URL",
			http.StatusInternalServerError,
		)
		return
	}

	host := parsedURL.Host

	var result string

	if protocol == "vless" {

		result = fmt.Sprintf(
			"vless://%s@%s:443?encryption=none&security=tls&type=ws&host=%s&path=%s&sni=%s#%s",
			clientUUID,
			host,
			url.QueryEscape(host),
			url.QueryEscape("/vless"),
			url.QueryEscape(host),
			url.QueryEscape(name),
		)

	} else {

		result = fmt.Sprintf(
			"trojan://%s@%s:443?security=tls&type=ws&host=%s&path=%s&sni=%s#%s",
			url.QueryEscape(password),
			host,
			url.QueryEscape(host),
			url.QueryEscape("/trojan"),
			url.QueryEscape(host),
			url.QueryEscape(name),
		)
	}

	w.Header().Set(
		"Content-Type",
		"text/plain; charset=utf-8",
	)

	_, _ = w.Write([]byte(result))
}

func parseQuotaGB(value string) (int64, error) {

	if value == "" {
		return 0, nil
	}

	gb, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}

	if gb < 0 {
		return 0, fmt.Errorf("quota cannot be negative")
	}

	const bytesPerGB = 1000000000.0

	bytes := gb * bytesPerGB

	if bytes > float64(^uint64(0)>>1) {
		return 0, fmt.Errorf("quota is too large")
	}

	return int64(bytes), nil
}

func generatePassword(length int) string {

	b := make([]byte, length)

	if _, err := rand.Read(b); err != nil {
		return uuid.New().String()
	}

	return hex.EncodeToString(b)[:length]
}
