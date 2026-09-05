package handlers

import (
	"log"
	"net/http"
	"panel/database"
	"time"
)

type DashboardData struct {
	Clients []ClientView
}

type ClientView struct {
	ID           int64
	Name         string
	Protocol     string
	TrafficUsed  int64
	TrafficLimit int64
	Enabled      bool
	Online       bool
	LastSeen     *time.Time
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := database.DB.Query(`
		SELECT
			id,
			name,
			protocol,
			traffic_used_bytes,
			traffic_limit_bytes,
			enabled,
			last_seen
		FROM clients
		ORDER BY id DESC
	`)
	if err != nil {
		log.Println("DASHBOARD DATABASE QUERY ERROR:", err)
		http.Error(w, "failed to load clients", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var clients []ClientView

	for rows.Next() {
		var (
			c           ClientView
			enabled     int
			lastSeenStr *string
		)

		if err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Protocol,
			&c.TrafficUsed,
			&c.TrafficLimit,
			&enabled,
			&lastSeenStr,
		); err != nil {
			log.Println("DASHBOARD ROW SCAN ERROR:", err)
			http.Error(w, "failed to read clients", http.StatusInternalServerError)
			return
		}

		c.Enabled = enabled == 1

		if lastSeenStr != nil && *lastSeenStr != "" {
			if t, err := parseDatabaseTime(*lastSeenStr); err == nil {
				c.LastSeen = &t
				c.Online = time.Since(t) <= 2*time.Minute
			} else {
				log.Println("DASHBOARD TIME PARSE ERROR:", err)
			}
		}

		clients = append(clients, c)
	}

	if err := rows.Err(); err != nil {
		log.Println("DASHBOARD ROWS ERROR:", err)
		http.Error(w, "failed to read clients", http.StatusInternalServerError)
		return
	}

	data := DashboardData{
		Clients: clients,
	}

	if err := renderTemplate(w, "templates/dashboard.html", data); err != nil {
		log.Println("DASHBOARD RENDER ERROR:", err)
		http.Error(w, "failed to render dashboard", http.StatusInternalServerError)
		return
	}
}

func parseDatabaseTime(value string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}

	var err error

	for _, layout := range layouts {
		var t time.Time

		t, err = time.Parse(layout, value)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, err
}
