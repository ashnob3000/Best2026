package database

import (
"database/sql"
"fmt"
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

// SQLite works best with a single writer connection.  
DB.SetMaxOpenConns(1)  
DB.SetMaxIdleConns(1)  

if err := DB.Ping(); err != nil {  
	return err  
}  

// Wait up to 5 seconds instead of immediately returning SQLITE_BUSY.  
if _, err := DB.Exec(`PRAGMA busy_timeout = 5000`); err != nil {  
	return fmt.Errorf("failed to set busy timeout: %w", err)  
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
if err != nil {  
	return err  
}  

columns := map[string]string{  
	"traffic_limit_bytes": "INTEGER NOT NULL DEFAULT 0",  
	"traffic_used_bytes":  "INTEGER NOT NULL DEFAULT 0",  
	"enabled":             "INTEGER NOT NULL DEFAULT 1",  
	"last_seen":           "DATETIME",  
}  

for name, definition := range columns {  
	exists, err := columnExists("clients", name)  
	if err != nil {  
		return err  
	}  

	if !exists {  
		query := fmt.Sprintf(  
			"ALTER TABLE clients ADD COLUMN %s %s",  
			name,  
			definition,  
		)  

		if _, err := DB.Exec(query); err != nil {  
			return err  
		}  
	}  
}  

return nil

}

func columnExists(table string, column string) (bool, error) {
rows, err := DB.Query("PRAGMA table_info(" + table + ")")
if err != nil {
return false, err
}
defer rows.Close()

for rows.Next() {  
	var (  
		cid      int  
		name     string  
		dataType string  
		notNull  int  
		defaultV interface{}  
		primary  int  
	)  

	if err := rows.Scan(  
		&cid,  
		&name,  
		&dataType,  
		&notNull,  
		&defaultV,  
		&primary,  
	); err != nil {  
		return false, err  
	}  

	if name == column {  
		return true, nil  
	}  
}  

return false, rows.Err()

}
