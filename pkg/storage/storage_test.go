package storage_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/censys/scan-takehome/pkg/storage"
)

var (
	testDB        *sql.DB
	testContainer testcontainers.Container
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		Env:          map[string]string{"POSTGRES_USER": "postgres", "POSTGRES_PASSWORD": "postgres", "POSTGRES_DB": "scans_test"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		fmt.Println("failed to start postgres container:", err)
		os.Exit(1)
	}

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("postgres://postgres:postgres@%s:%s/scans_test?sslmode=disable", host, port.Port())

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fmt.Println("failed to open db:", err)
		os.Exit(1)
	}

	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Apply the migration to the test db
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "../../")
	migrationPath := filepath.Join(root, "db/migrations/0001_init.sql")

	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		fmt.Println("failed to read migration:", err)
		os.Exit(1)
	}
	_, err = db.ExecContext(ctx, string(migrationSQL))
	if err != nil {
		fmt.Println("failed to apply migration:", err)
		os.Exit(1)
	}

	testDB = db
	testContainer = container

	exitCode := m.Run()

	db.Close()
	container.Terminate(ctx)

	os.Exit(exitCode)
}

func TestStoreScanRecord(t *testing.T) {
	ctx := context.Background()
	store := storage.NewPostgresStore(testDB)

	// Fresh insert should work
	record := storage.ScanRecord{
		Ip:          "1.1.1.42",
		Port:        8080,
		Service:     "HTTP",
		Timestamp:   100,
		ResponseStr: "initial",
	}
	if err := store.StoreScanRecord(ctx, record); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Same timestamp should be ignored
	if err := store.StoreScanRecord(ctx, record); err != nil {
		t.Fatalf("stale write: %v", err)
	}

	// Newer timestamp should update the row
	record.Timestamp = 200
	record.ResponseStr = "updated"
	if err := store.StoreScanRecord(ctx, record); err != nil {
		t.Fatalf("update: %v", err)
	}

	var ts int64
	var resp string
	err := testDB.QueryRow(
		`SELECT timestamp, response_str FROM scan_records WHERE ip=$1 AND port=$2 AND service=$3`,
		record.Ip, record.Port, record.Service,
	).Scan(&ts, &resp)
	if err != nil {
		t.Fatalf("select row: %v", err)
	}
	if ts != 200 || resp != "updated" {
		t.Fatalf("unexpected row values: ts=%d resp=%q", ts, resp)
	}
}
