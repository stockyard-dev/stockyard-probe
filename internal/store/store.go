package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct { db *sql.DB }

type Request struct {
	ID           string   `json:"id"`
	Method       string   `json:"method"`
	URL          string   `json:"url"`
	StatusCode   int      `json:"status_code"`
	Duration     string   `json:"duration"`
	Body         string   `json:"body"`
	CreatedAt    string   `json:"created_at"`
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	dsn := filepath.Join(dataDir, "probe.db") + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS requests (
			id TEXT PRIMARY KEY,\n\t\t\tmethod TEXT DEFAULT '',\n\t\t\turl TEXT DEFAULT '',\n\t\t\tstatus_code INTEGER DEFAULT 0,\n\t\t\tduration TEXT DEFAULT '',\n\t\t\tbody TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now'))
		)`)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }

func (d *DB) Create(e *Request) error {
	e.ID = genID()
	e.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	_, err := d.db.Exec(`INSERT INTO requests (id, method, url, status_code, duration, body, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Method, e.URL, e.StatusCode, e.Duration, e.Body, e.CreatedAt)
	return err
}

func (d *DB) Get(id string) *Request {
	row := d.db.QueryRow(`SELECT id, method, url, status_code, duration, body, created_at FROM requests WHERE id=?`, id)
	var e Request
	if err := row.Scan(&e.ID, &e.Method, &e.URL, &e.StatusCode, &e.Duration, &e.Body, &e.CreatedAt); err != nil {
		return nil
	}
	return &e
}

func (d *DB) List() []Request {
	rows, err := d.db.Query(`SELECT id, method, url, status_code, duration, body, created_at FROM requests ORDER BY created_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []Request
	for rows.Next() {
		var e Request
		if err := rows.Scan(&e.ID, &e.Method, &e.URL, &e.StatusCode, &e.Duration, &e.Body, &e.CreatedAt); err != nil {
			continue
		}
		result = append(result, e)
	}
	return result
}

func (d *DB) Delete(id string) error {
	_, err := d.db.Exec(`DELETE FROM requests WHERE id=?`, id)
	return err
}

func (d *DB) Count() int {
	var n int
	d.db.QueryRow(`SELECT COUNT(*) FROM requests`).Scan(&n)
	return n
}
