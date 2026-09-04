package handlers

import (
	"net/http"
	"panel/database"
)

type DashboardData struct {
	Clients []ClientView
}

type ClientView struct {
	ID       int64
	Name     string
	Protocol string
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, name, protocol
		FROM clients
		ORDER BY id DESC
	`)
	if err != nil {
		http.Error(w, "failed to load clients", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var clients []ClientView

	for rows.Next() {
		var c ClientView

		if err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Protocol,
		); err != nil {
			http.Error(w, "failed to read clients", http.StatusInternalServerError)
			return
		}

		clients = append(clients, c)
	}

	data := DashboardData{
		Clients: clients,
	}

	if err := renderTemplate(w, "templates/dashboard.html", data); err != nil {
		http.Error(w, "failed to render dashboard", http.StatusInternalServerError)
		return
	}
}
