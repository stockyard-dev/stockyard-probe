package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ db *sql.DB }

type Bin struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	ResponseCode int    `json:"response_code"`
	ResponseBody string `json:"response_body,omitempty"`
	ResponseType string `json:"response_type,omitempty"`
	CreatedAt    string `json:"created_at"`
	RequestCount int    `json:"request_count"`
}

type Request struct {
	ID        string            `json:"id"`
	BinID     string            `json:"bin_id"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Query     string            `json:"query,omitempty"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body,omitempty"`
	IP        string            `json:"ip,omitempty"`
	Size      int               `json:"size"`
	CreatedAt string            `json:"created_at"`
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
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS bins (id TEXT PRIMARY KEY, name TEXT NOT NULL, slug TEXT UNIQUE NOT NULL, response_code INTEGER DEFAULT 200, response_body TEXT DEFAULT '', response_type TEXT DEFAULT 'application/json', created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE TABLE IF NOT EXISTS requests (id TEXT PRIMARY KEY, bin_id TEXT NOT NULL REFERENCES bins(id) ON DELETE CASCADE, method TEXT DEFAULT 'GET', path TEXT DEFAULT '/', query TEXT DEFAULT '', headers_json TEXT DEFAULT '{}', body TEXT DEFAULT '', ip TEXT DEFAULT '', size INTEGER DEFAULT 0, created_at TEXT DEFAULT (datetime('now')))`,
		`CREATE INDEX IF NOT EXISTS idx_requests_bin ON requests(bin_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bins_slug ON bins(slug)`,
	} {
		if _, err := db.Exec(q); err != nil {
			return nil, fmt.Errorf("migrate: %w", err)
		}
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS extras(resource TEXT NOT NULL,record_id TEXT NOT NULL,data TEXT NOT NULL DEFAULT '{}',PRIMARY KEY(resource, record_id))`)
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }
func genID() string        { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string          { return time.Now().UTC().Format(time.RFC3339) }

func (d *DB) CreateBin(b *Bin) error {
	b.ID = genID()
	b.CreatedAt = now()
	if b.ResponseCode <= 0 {
		b.ResponseCode = 200
	}
	if b.ResponseType == "" {
		b.ResponseType = "application/json"
	}
	_, err := d.db.Exec(`INSERT INTO bins (id,name,slug,response_code,response_body,response_type,created_at) VALUES (?,?,?,?,?,?,?)`,
		b.ID, b.Name, b.Slug, b.ResponseCode, b.ResponseBody, b.ResponseType, b.CreatedAt)
	return err
}

func (d *DB) GetBin(id string) *Bin {
	var b Bin
	if err := d.db.QueryRow(`SELECT id,name,slug,response_code,response_body,response_type,created_at FROM bins WHERE id=?`, id).Scan(&b.ID, &b.Name, &b.Slug, &b.ResponseCode, &b.ResponseBody, &b.ResponseType, &b.CreatedAt); err != nil {
		return nil
	}
	d.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE bin_id=?`, id).Scan(&b.RequestCount)
	return &b
}

func (d *DB) GetBinBySlug(slug string) *Bin {
	var b Bin
	if err := d.db.QueryRow(`SELECT id,name,slug,response_code,response_body,response_type,created_at FROM bins WHERE slug=?`, slug).Scan(&b.ID, &b.Name, &b.Slug, &b.ResponseCode, &b.ResponseBody, &b.ResponseType, &b.CreatedAt); err != nil {
		return nil
	}
	d.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE bin_id=?`, b.ID).Scan(&b.RequestCount)
	return &b
}

func (d *DB) ListBins() []Bin {
	rows, _ := d.db.Query(`SELECT id,name,slug,response_code,response_body,response_type,created_at FROM bins ORDER BY created_at DESC`)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var out []Bin
	for rows.Next() {
		var b Bin
		rows.Scan(&b.ID, &b.Name, &b.Slug, &b.ResponseCode, &b.ResponseBody, &b.ResponseType, &b.CreatedAt)
		d.db.QueryRow(`SELECT COUNT(*) FROM requests WHERE bin_id=?`, b.ID).Scan(&b.RequestCount)
		out = append(out, b)
	}
	return out
}

func (d *DB) UpdateBin(id string, b *Bin) error {
	_, err := d.db.Exec(`UPDATE bins SET name=?,response_code=?,response_body=?,response_type=? WHERE id=?`,
		b.Name, b.ResponseCode, b.ResponseBody, b.ResponseType, id)
	return err
}

func (d *DB) DeleteBin(id string) error {
	d.db.Exec(`DELETE FROM requests WHERE bin_id=?`, id)
	_, err := d.db.Exec(`DELETE FROM bins WHERE id=?`, id)
	return err
}

func (d *DB) CaptureRequest(r *Request) error {
	r.ID = genID()
	r.CreatedAt = now()
	if r.Headers == nil {
		r.Headers = map[string]string{}
	}
	hj, _ := json.Marshal(r.Headers)
	_, err := d.db.Exec(`INSERT INTO requests (id,bin_id,method,path,query,headers_json,body,ip,size,created_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		r.ID, r.BinID, r.Method, r.Path, r.Query, string(hj), r.Body, r.IP, r.Size, r.CreatedAt)
	if err == nil {
		d.db.Exec(`DELETE FROM requests WHERE bin_id=? AND id NOT IN (SELECT id FROM requests WHERE bin_id=? ORDER BY created_at DESC LIMIT 500)`, r.BinID, r.BinID)
	}
	return err
}

func (d *DB) ListRequests(binID string, limit int) []Request {
	if limit <= 0 {
		limit = 50
	}
	rows, _ := d.db.Query(`SELECT id,bin_id,method,path,query,headers_json,body,ip,size,created_at FROM requests WHERE bin_id=? ORDER BY created_at DESC LIMIT ?`, binID, limit)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var out []Request
	for rows.Next() {
		var r Request
		var hj string
		rows.Scan(&r.ID, &r.BinID, &r.Method, &r.Path, &r.Query, &hj, &r.Body, &r.IP, &r.Size, &r.CreatedAt)
		json.Unmarshal([]byte(hj), &r.Headers)
		out = append(out, r)
	}
	return out
}

func (d *DB) GetRequest(id string) *Request {
	var r Request
	var hj string
	if err := d.db.QueryRow(`SELECT id,bin_id,method,path,query,headers_json,body,ip,size,created_at FROM requests WHERE id=?`, id).Scan(&r.ID, &r.BinID, &r.Method, &r.Path, &r.Query, &hj, &r.Body, &r.IP, &r.Size, &r.CreatedAt); err != nil {
		return nil
	}
	json.Unmarshal([]byte(hj), &r.Headers)
	return &r
}

func (d *DB) ClearRequests(binID string) error {
	_, err := d.db.Exec(`DELETE FROM requests WHERE bin_id=?`, binID)
	return err
}

type Stats struct {
	Bins     int `json:"bins"`
	Requests int `json:"requests"`
}

func (d *DB) Stats() Stats {
	var s Stats
	d.db.QueryRow(`SELECT COUNT(*) FROM bins`).Scan(&s.Bins)
	d.db.QueryRow(`SELECT COUNT(*) FROM requests`).Scan(&s.Requests)
	return s
}

// ─── Extras: generic key-value storage for personalization custom fields ───

func (d *DB) GetExtras(resource, recordID string) string {
	var data string
	err := d.db.QueryRow(
		`SELECT data FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	).Scan(&data)
	if err != nil || data == "" {
		return "{}"
	}
	return data
}

func (d *DB) SetExtras(resource, recordID, data string) error {
	if data == "" {
		data = "{}"
	}
	_, err := d.db.Exec(
		`INSERT INTO extras(resource, record_id, data) VALUES(?, ?, ?)
		 ON CONFLICT(resource, record_id) DO UPDATE SET data=excluded.data`,
		resource, recordID, data,
	)
	return err
}

func (d *DB) DeleteExtras(resource, recordID string) error {
	_, err := d.db.Exec(
		`DELETE FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	)
	return err
}

func (d *DB) AllExtras(resource string) map[string]string {
	out := make(map[string]string)
	rows, _ := d.db.Query(
		`SELECT record_id, data FROM extras WHERE resource=?`,
		resource,
	)
	if rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, data string
		rows.Scan(&id, &data)
		out[id] = data
	}
	return out
}
