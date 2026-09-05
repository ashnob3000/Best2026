package xray

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"panel/database"
)

type statsResponse struct {
	Stat []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"stat"`
}

type TrafficCollector struct {
	manager *Manager

	mu sync.Mutex

	previous map[int64]int64
}

func NewTrafficCollector(manager *Manager) *TrafficCollector {
	return &TrafficCollector{
		manager:  manager,
		previous: make(map[int64]int64),
	}
}

func (tc *TrafficCollector) Start() {
	go func() {

		// Give Xray a moment to start.
		time.Sleep(5 * time.Second)

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			tc.collect()

			<-ticker.C
		}
	}()
}

func (tc *TrafficCollector) collect() {

	rows, err := database.DB.Query(`
		SELECT id
		FROM clients
		WHERE enabled = 1
	`)
	if err != nil {
		log.Println("traffic collector database error:", err)
		return
	}

	defer rows.Close()

	for rows.Next() {

		var id int64

		if err := rows.Scan(&id); err != nil {
			continue
		}

		up, down, err := queryUserTraffic(id)

		if err != nil {
			continue
		}

		current := up + down

		tc.mu.Lock()

		previous := tc.previous[id]

		var delta int64

		if current >= previous {
			delta = current - previous
		} else {
			// Xray restarted and its counters reset.
			delta = current
		}

		tc.previous[id] = current

		tc.mu.Unlock()

		if delta > 0 {
			tc.applyTraffic(id, delta)
		}
	}
}

func queryUserTraffic(id int64) (int64, int64, error) {

	email := fmt.Sprintf("client-%d", id)

	cmd := exec.Command(
		"xray",
		"api",
		"statsquery",
		"--server=127.0.0.1:10085",
		"-pattern",
		"user>>>"+email+">>>traffic>>>",
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	var response statsResponse

	if err := json.Unmarshal(output, &response); err != nil {
		return 0, 0, err
	}

	var uplink int64
	var downlink int64

	for _, stat := range response.Stat {

		value, err := strconv.ParseInt(
			strings.TrimSpace(stat.Value),
			10,
			64,
		)
		if err != nil {
			continue
		}

		if strings.HasSuffix(stat.Name, ">>>uplink") {
			uplink = value
		}

		if strings.HasSuffix(stat.Name, ">>>downlink") {
			downlink = value
		}
	}

	return uplink, downlink, nil
}

func (tc *TrafficCollector) applyTraffic(id int64, delta int64) {

	var (
		used  int64
		limit int64
		enabled int
	)

	err := database.DB.QueryRow(`
		SELECT traffic_used_bytes,
		       traffic_limit_bytes,
		       enabled
		FROM clients
		WHERE id = ?
	`, id).Scan(
		&used,
		&limit,
		&enabled,
	)

	if err != nil {
		return
	}

	if enabled == 0 {
		return
	}

	newUsed := used + delta

	_, err = database.DB.Exec(`
		UPDATE clients
		SET traffic_used_bytes = ?,
		    last_seen = CURRENT_TIMESTAMP
		WHERE id = ?
	`,
		newUsed,
		id,
	)

	if err != nil {
		log.Println("failed to update traffic:", err)
		return
	}

	// 0 means unlimited.
	if limit > 0 && newUsed >= limit {

		log.Printf(
			"Client %d reached traffic limit. Disabling client.",
			id,
		)

		_, err := database.DB.Exec(`
			UPDATE clients
			SET enabled = 0
			WHERE id = ?
		`, id)

		if err != nil {
			log.Println("failed to disable client:", err)
			return
		}

		// Remove the user from Xray immediately.
		if tc.manager != nil {
			if err := tc.manager.Reload(); err != nil {
				log.Println("failed to reload Xray after quota:", err)
			}
		}
	}
}
