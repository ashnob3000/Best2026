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

type ClientHandler struct {
	TunnelManager interface {
		VLESSURL() string
		TrojanURL() string
	}
}

func NewClientHandler(tunnelManager interface {
	VLESSURL() string
	TrojanURL() string
}) *ClientHandler {
	return &ClientHandler{
		TunnelManager: tunnelManager,
	}
}

func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	protocol := strings.TrimSpace(r.FormValue("protocol"))

	if name == "" {
		http.Error(w, "client name is required", http.StatusBadRequest)
		return
	}

	if protocol != "vless" && protocol != "trojan" {
		http.Error(w, "invalid protocol", http.StatusBadRequest)
		return
	}

	var clientUUID string
	var password string

	if protocol == "vless" {
		clientUUID = uuid.New().String()
	} else {
		password = generatePassword(24)
	}

	_, err := database.DB.Exec(`
		INSERT INTO clients
		(name, protocol, uuid, password, created_at)
		VALUES (?, ?, ?, ?, ?)
	`,
		name,
		protocol,
		clientUUID,
		password,
		time.Now(),
	)

	if err != nil {
		http.Error(w, "failed to create client", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

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

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

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

	var name string
	var protocol string
	var clientUUID string
	var password string

	err = database.DB.QueryRow(`
		SELECT name, protocol, uuid, password
		FROM clients
		WHERE id = ?
	`, id).Scan(
		&name,
		&protocol,
		&clientUUID,
		&password,
	)

	if err != nil {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	if h.TunnelManager == nil {
		http.Error(w, "tunnel manager unavailable", http.StatusServiceUnavailable)
		return
	}

	var tunnelURL string

	if protocol == "vless" {
		tunnelURL = h.TunnelManager.VLESSURL()
	} else {
		tunnelURL = h.TunnelManager.TrojanURL()
	}

	if tunnelURL == "" {
		http.Error(w, "tunnel is not ready yet", http.StatusServiceUnavailable)
		return
	}

	parsedURL, err := url.Parse(tunnelURL)
	if err != nil {
		http.Error(w, "invalid tunnel URL", http.StatusInternalServerError)
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

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(result))
}

func generatePassword(length int) string {
	b := make([]byte, length)

	if _, err := rand.Read(b); err != nil {
		return uuid.New().String()
	}

	return hex.EncodeToString(b)[:length]
}
