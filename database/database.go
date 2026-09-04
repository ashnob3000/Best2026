package database

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init() error {
	if err := os.MkdirAll("data", 0755); err != nil {
		return err
	}

	var err error

	DB, err = sql.Open("sqlite", "data/panel.db")
	if err != nil {
		return err
	}

	if err := DB.Ping(); err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS clients (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			protocol TEXT NOT NULL,
			uuid TEXT,
			password TEXT,
			created_at DATETIME NOT NULL
		)
	`)

	return err
}
