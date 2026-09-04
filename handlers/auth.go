package handlers

import (
	"net/http"
	"os"
	"time"
)

var sessionCookieName = "panel_session"

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		http.ServeFile(w, r, "templates/login.html")
		return
	}
	user := os.Getenv("PANEL_USER")
	pass := os.Getenv("PANEL_PASS")
	if user == "" { user = "admin" }
	if pass == "" { pass = "admin123" }

	r.ParseForm()
	if r.FormValue("username") == user && r.FormValue("password") == pass {
		http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "authenticated", Expires: time.Now().Add(24 * time.Hour), Path: "/"})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", MaxAge: -1, Path: "/"})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func IsAuthenticated(r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	return err == nil && c.Value == "authenticated"
}
