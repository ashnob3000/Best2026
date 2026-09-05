package xray

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"panel/database"
)

type statsResponse struct {
	Stat []struct {
		Name  string `json:"name"`
		Value int64  `json:"value"`
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

		log.Println("TRAFFIC COLLECTOR: started")

		time.Sleep(5 * time.Second)

		log.Println("TRAFFIC COLLECTOR: starting collection loop")

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			tc.collect()

			<-ticker.C
		}
	}()
}

func (tc *TrafficCollector) collect() {

	log.Println("TRAFFIC COLLECTOR: collect() running")

	// اول فقط ID کلاینت‌ها را می‌خوانیم.
	// مهم: rows باید قبل از هر UPDATE/Query بعدی بسته شود،
	// چون SQLite با یک connection کار می‌کند.
	rows, err := database.DB.Query(`
		SELECT id
		FROM clients
		WHERE enabled = 1
	`)
	if err != nil {
		log.Println("TRAFFIC COLLECTOR: database error:", err)
		return
	}

	var clientIDs []int64

	for rows.Next() {

		var id int64

		if err := rows.Scan(&id); err != nil {
			log.Println("TRAFFIC COLLECTOR: failed to scan client:", err)
			continue
		}

		clientIDs = append(clientIDs, id)
	}

	if err := rows.Err(); err != nil {
		log.Println("TRAFFIC COLLECTOR: rows error:", err)
	}

	rows.Close()

	log.Printf(
		"TRAFFIC COLLECTOR: collection found %d client(s)",
		len(clientIDs),
	)

	// حالا که rows کاملاً بسته شده، می‌توانیم آزادانه
	// Query و UPDATE انجام دهیم.
	for _, id := range clientIDs {

		log.Printf(
			"TRAFFIC COLLECTOR: querying client-%d",
			id,
		)

		up, down, err := queryUserTraffic(id)

		if err != nil {
			log.Printf(
				"TRAFFIC COLLECTOR: stats query failed for client-%d: %v",
				id,
				err,
			)
			continue
		}

		log.Printf(
			"TRAFFIC COLLECTOR: client-%d uplink=%d downlink=%d",
			id,
			up,
			down,
		)

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

		tc.mu.Unlock()

		log.Printf(
			"TRAFFIC COLLECTOR: client-%d current=%d previous=%d delta=%d",
			id,
			current,
			previous,
			delta,
		)

		if delta > 0 {

			// فقط اگر ثبت در دیتابیس موفق شد،
			// previous را به‌روزرسانی می‌کنیم.
			if err := tc.applyTraffic(id, delta); err != nil {
				log.Printf(
					"TRAFFIC COLLECTOR: failed to apply traffic for client-%d: %v",
					id,
					err,
				)
				continue
			}
		}

		tc.mu.Lock()
		tc.previous[id] = current
		tc.mu.Unlock()
	}

	log.Printf(
		"TRAFFIC COLLECTOR: collection finished, clients=%d",
		len(clientIDs),
	)
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

	output, err := cmd.CombinedOutput()

	log.Printf(
		"STATS QUERY client-%d: err=%v output=%s",
		id,
		err,
		strings.TrimSpace(string(output)),
	)

	if err != nil {
		return 0, 0, err
	}

	var response statsResponse

	if err := json.Unmarshal(output, &response); err != nil {
		log.Printf(
			"STATS QUERY client-%d: JSON parse error: %v",
			id,
			err,
		)
		return 0, 0, err
	}

	var uplink int64
	var downlink int64

	for _, stat := range response.Stat {

		value := stat.Value

		if strings.HasSuffix(stat.Name, ">>>uplink") {
			uplink = value
		}

		if strings.HasSuffix(stat.Name, ">>>downlink") {
			downlink = value
		}
	}

	return uplink, downlink, nil
}

func (tc *TrafficCollector) applyTraffic(id int64, delta int64) error {

	var (
		used    int64
		limit   int64
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
		log.Printf(
			"TRAFFIC COLLECTOR: failed to read client-%d: %v",
			id,
			err,
		)
		return err
	}

	if enabled == 0 {
		return nil
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
		return err
	}

	log.Printf(
		"TRAFFIC: client-%d added=%d total=%d limit=%d",
		id,
		delta,
		newUsed,
		limit,
	)

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
			return err
		}

		if tc.manager != nil {
			if err := tc.manager.Reload(); err != nil {
				log.Println("failed to reload Xray after quota:", err)
				return err
			}
		}
	}

	return nil
}
