package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"sync"
	"time"
)

var sessionCookieName = "panel_session"

const sessionDuration = 24 * time.Hour

var (
	sessions   = make(map[string]time.Time)
	sessionsMu sync.RWMutex
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {
		http.ServeFile(w, r, "templates/login.html")
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := os.Getenv("PANEL_USER")
	pass := os.Getenv("PANEL_PASS")

	if user == "" {
		user = "admin"
	}

	if pass == "" {
		pass = "admin123"
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	if r.FormValue("username") != user ||
		r.FormValue("password") != pass {
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}

	sessionID, err := generateSessionID()
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	expires := time.Now().Add(sessionDuration)

	sessionsMu.Lock()
	sessions[sessionID] = expires
	sessionsMu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Expires:  expires,
		MaxAge:   int(sessionDuration.Seconds()),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})

	http.Redirect(w, r, "/?refresh="+time.Now().Format("20060102150405.000000000"), http.StatusSeeOther)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie(sessionCookieName)

	if err == nil && cookie.Value != "" {
		sessionsMu.Lock()
		delete(sessions, cookie.Value)
		sessionsMu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func IsAuthenticated(r *http.Request) bool {

	cookie, err := r.Cookie(sessionCookieName)

	if err != nil || cookie.Value == "" {
		return false
	}

	sessionsMu.RLock()
	expires, exists := sessions[cookie.Value]
	sessionsMu.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(expires) {
		sessionsMu.Lock()
		delete(sessions, cookie.Value)
		sessionsMu.Unlock()

		return false
	}

	return true
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		if !IsAuthenticated(r) {

			if r.Method == http.MethodGet {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func generateSessionID() (string, error) {

	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
