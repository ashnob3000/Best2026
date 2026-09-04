package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"panel/database"
)

type ClientHandler struct{}

func NewClientHandler() *ClientHandler {
	return &ClientHandler{}
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

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if protocol == "vless" {
		_, _ = w.Write([]byte(
			"VLESS CLIENT\n" +
				"Name: " + name + "\n" +
				"UUID: " + clientUUID + "\n",
		))
		return
	}

	_, _ = w.Write([]byte(
		"TROJAN CLIENT\n" +
			"Name: " + name + "\n" +
			"Password: " + password + "\n",
	))
}

func generatePassword(length int) string {
	b := make([]byte, length)

	if _, err := rand.Read(b); err != nil {
		return uuid.New().String()
	}

	return hex.EncodeToString(b)[:length]
}
