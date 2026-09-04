package handlers

import "net/http"

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "templates/dashboard.html")
}
