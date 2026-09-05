package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"panel/config"
	"panel/database"
)

type Manager struct {
	mu  sync.Mutex
	cmd *exec.Cmd
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.generateConfigLocked(); err != nil {
		return err
	}

	return m.startLocked()
}

func (m *Manager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		log.Println("Stopping Xray...")

		_ = m.cmd.Process.Kill()
		_, _ = m.cmd.Process.Wait()

		m.cmd = nil
	}

	if err := m.generateConfigLocked(); err != nil {
		return err
	}

	log.Println("Starting Xray with updated configuration...")

	return m.startLocked()
}

func (m *Manager) startLocked() error {
	cmd := exec.CommandContext(
		context.Background(),
		"xray",
		"run",
		"-c",
		"config/xray.json",
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Xray: %w", err)
	}

	m.cmd = cmd

	log.Println("Xray started")

	return nil
}

func (m *Manager) generateConfigLocked() error {

	rows, err := database.DB.Query(`
		SELECT id, uuid, password, enabled
		FROM clients
		ORDER BY id
	`)
	if err != nil {
		return fmt.Errorf("failed to query clients: %w", err)
	}

	defer rows.Close()

	var clients []config.Client

	for rows.Next() {

		var (
			id       int64
			uuid     string
			password string
			enabled  int
		)

		if err := rows.Scan(
			&id,
			&uuid,
			&password,
			&enabled,
		); err != nil {
			return fmt.Errorf("failed to read client: %w", err)
		}

		clients = append(clients, config.Client{
			ID:       id,
			UUID:     uuid,
			Password: password,
			Email:    fmt.Sprintf("client-%d", id),
			Enabled:  enabled == 1,
		})
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed while reading clients: %w", err)
	}

	data, err := config.GenerateXrayConfig(clients)
	if err != nil {
		return fmt.Errorf("failed to generate Xray config: %w", err)
	}

	var pretty json.RawMessage

	if err := json.Unmarshal(data, &pretty); err != nil {
		return fmt.Errorf("invalid generated Xray config: %w", err)
	}

	if err := os.WriteFile(
		"config/xray.json",
		data,
		0644,
	); err != nil {
		return fmt.Errorf("failed to write Xray config: %w", err)
	}

	enabledCount := 0

	for _, client := range clients {
		if client.Enabled {
			enabledCount++
		}
	}

	log.Printf(
		"Xray configuration updated: %d client(s), %d enabled",
		len(clients),
		enabledCount,
	)

	return nil
}
