package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pudymody/pinger-go/endpoint"
	"github.com/pudymody/pinger-go/hit"

	_ "github.com/mattn/go-sqlite3"
)

type Sqlite struct {
	conn *sql.DB
}

func NewSqlite(ctx context.Context, filepath string) (*Sqlite, error) {
	dbPath := fmt.Sprintf("%s?_timeout=5000&_journal_mode=wal&_sync=1&_cache_size=20000&_fk=0&_auto_vacuum=2", filepath)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS endpoints (
		id INTEGER PRIMARY KEY,
		domain TEXT,
		code_ok INTEGER,
		timeout string,
		interval string
	);

	CREATE TABLE IF NOT EXISTS hits (
		endpoint_id INTEGER,
		status TEXT,
		latency INTEGER,
		created_at DATETIME 
	);	`)

	if err != nil {
		return nil, err
	}

	_, err = db.ExecContext(ctx, `
		-- Store temporary tables and data in memory for better performance
		PRAGMA temp_store = MEMORY;

		-- Set the mmap_size to 2GB for faster read/write access using memory-mapped I/O
		PRAGMA mmap_size = 2147483648;

		-- Set the page size to 8KB for balanced memory usage and performance
		PRAGMA page_size = 8192;
	`)

	if err != nil {
		return nil, err
	}

	return &Sqlite{
		conn: db,
	}, nil
}

func (s *Sqlite) InsertEndpoint(ctx context.Context, item endpoint.Endpoint) error {
	_, err := s.conn.ExecContext(ctx, "INSERT INTO endpoints (domain, code_ok, timeout,interval) VALUES (?,?,?,?)", item.Domain, item.CodeOK, item.Timeout.String(), item.Interval.String())
	return err
}

func (s *Sqlite) GetEndpoint(ctx context.Context, id int64) (endpoint.Endpoint, error) {
	rows := s.conn.QueryRowContext(ctx, "SELECT id, domain, code_ok, timeout, interval FROM endpoints WHERE id = ?", id)

	var item endpoint.Endpoint
	var durationString string
	var intervalString string

	if err := rows.Scan(&item.ID, &item.Domain, &item.CodeOK, &durationString, &intervalString); err != nil {
		return endpoint.Endpoint{}, err
	}

	timeout, err := time.ParseDuration(durationString)
	if err != nil {
		return endpoint.Endpoint{}, err
	}
	item.Timeout = timeout

	interval, err := time.ParseDuration(intervalString)
	if err != nil {
		return endpoint.Endpoint{}, err
	}
	item.Interval = interval

	return item, nil
}

func (s *Sqlite) GetAllEndpoints(ctx context.Context) ([]endpoint.Endpoint, error) {
	rows, err := s.conn.QueryContext(ctx, "SELECT id, domain, code_ok, timeout, interval FROM endpoints")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]endpoint.Endpoint, 0)

	for rows.Next() {
		var item endpoint.Endpoint
		var durationString string
		var intervalString string

		if err := rows.Scan(&item.ID, &item.Domain, &item.CodeOK, &durationString, &intervalString); err != nil {
			return nil, err
		}

		timeout, err := time.ParseDuration(durationString)
		if err != nil {
			return nil, err
		}
		item.Timeout = timeout
		interval, err := time.ParseDuration(intervalString)
		if err != nil {
			return nil, err
		}
		item.Interval = interval
		items = append(items, item)
	}
	// If the database is being written to ensure to check for Close
	// errors that may be returned from the driver. The query may
	// encounter an auto-commit error and be forced to rollback changes.
	rerr := rows.Close()
	if rerr != nil {
		return nil, rerr
	}

	// Rows.Err will report the last error encountered by Rows.Scan.
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *Sqlite) UpdateEndpoint(ctx context.Context, item endpoint.Endpoint) error {
	_, err := s.conn.ExecContext(ctx, "UPDATE endpoints SET domain = ?, code_ok = ?, timeout = ?, interval = ? WHERE id = ?", item.Domain, item.CodeOK, item.Timeout.String(), item.Interval.String(), item.ID)
	return err
}

func (s *Sqlite) InsertHit(ctx context.Context, item hit.Hit) error {
	_, err := s.conn.ExecContext(ctx, "INSERT INTO hits (endpoint_id, status, latency, created_at) VALUES (?,?,?,?)", item.EndpointID, item.Status, item.Latency, item.CreatedAt)
	return err
}

func (s *Sqlite) GetHits(ctx context.Context, endpointID int64, from time.Time, to time.Time) ([]hit.Hit, error) {
	rows, err := s.conn.QueryContext(ctx, "SELECT endpoint_id, status, latency, created_at FROM hits WHERE created_at >= ? AND created_at <= ? ORDER BY created_at ASC", from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]hit.Hit, 0)

	for rows.Next() {
		var item hit.Hit

		if err := rows.Scan(&item.EndpointID, &item.Status, &item.Latency, &item.CreatedAt); err != nil {
			return nil, err
		}

		items = append(items, item)
	}
	// If the database is being written to ensure to check for Close
	// errors that may be returned from the driver. The query may
	// encounter an auto-commit error and be forced to rollback changes.
	rerr := rows.Close()
	if rerr != nil {
		return nil, rerr
	}

	// Rows.Err will report the last error encountered by Rows.Scan.
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
