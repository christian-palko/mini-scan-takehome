package storage

import (
	"context"
	"database/sql"
	"log"
)

type ScanRecord struct {
	Ip          string `json:"ip"`
	Port        uint32 `json:"port"`
	Service     string `json:"service"`
	Timestamp   int64  `json:"timestamp"`
	ResponseStr string `json:"response_str"`
}

type Store interface {
	StoreScanRecord(ctx context.Context, scanRecord ScanRecord) error
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	if db == nil {
		panic("storage: nil db")
	}
	return &PostgresStore{db: db}
}

func (s *PostgresStore) StoreScanRecord(ctx context.Context, scanRecord ScanRecord) error {
	const upsert = `
INSERT INTO scan_records (ip, port, service, timestamp, response_str)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (ip, port, service)
DO UPDATE SET
	timestamp = EXCLUDED.timestamp,
	response_str = EXCLUDED.response_str
WHERE EXCLUDED.timestamp > scan_records.timestamp
-- xmax is 0 for freshly inserted rows and non-zero after an update
RETURNING CASE WHEN xmax = 0 THEN 'inserted' ELSE 'updated' END`

	var action string
	err := s.db.QueryRowContext(ctx, upsert,
		scanRecord.Ip,
		scanRecord.Port,
		scanRecord.Service,
		scanRecord.Timestamp,
		scanRecord.ResponseStr,
	).Scan(&action)
	switch err {
	case nil:
		log.Printf("storage: %s scan for %s:%d/%s (ts=%d)", action, scanRecord.Ip, scanRecord.Port, scanRecord.Service, scanRecord.Timestamp)
		return nil
	case sql.ErrNoRows:
		log.Printf("storage: ignored stale scan for %s:%d/%s (ts=%d)", scanRecord.Ip, scanRecord.Port, scanRecord.Service, scanRecord.Timestamp)
		return nil
	default:
		return err
	}
}
