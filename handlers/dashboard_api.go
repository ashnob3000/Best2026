package handlers

import (
	"encoding/json"
	"net/http"
	"panel/database"
	"time"
)

type ClientStatus struct {
	ID            int64  `json:"id"`
	TrafficUsed   int64  `json:"traffic_used"`
	TrafficLimit  int64  `json:"traffic_limit"`
	Enabled       bool   `json:"enabled"`
	Online        bool   `json:"online"`
}

func ClientsStatusHandler(w http.ResponseWriter, r *http.Request) {

	if !IsAuthenticated(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := database.DB.Query(`
		SELECT
			id,
			traffic_used_bytes,
			traffic_limit_bytes,
			enabled,
			last_seen
		FROM clients
		ORDER BY id DESC
	`)

	if err != nil {
		http.Error(w, "failed to load client status", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var clients []ClientStatus

	for rows.Next() {

		var (
			c           ClientStatus
			enabled     int
			lastSeenStr *string
		)

		if err := rows.Scan(
			&c.ID,
			&c.TrafficUsed,
			&c.TrafficLimit,
			&enabled,
			&lastSeenStr,
		); err != nil {
			http.Error(w, "failed to read client status", http.StatusInternalServerError)
			return
		}

		c.Enabled = enabled == 1

		if lastSeenStr != nil && *lastSeenStr != "" {

			if t, err := parseDatabaseTime(*lastSeenStr); err == nil {
				c.Online = time.Since(t) <= 10*time.Second
			}
		}

		clients = append(clients, c)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "failed to read client status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(clients)
}
