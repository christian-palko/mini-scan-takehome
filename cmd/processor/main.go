package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/censys/scan-takehome/pkg/scanning"
	"github.com/censys/scan-takehome/pkg/storage"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func processMessage(ctx context.Context, msg *pubsub.Message, store storage.Store) {
	type scanEnvelope struct {
		Ip          string          `json:"ip"`
		Port        uint32          `json:"port"`
		Service     string          `json:"service"`
		Timestamp   int64           `json:"timestamp"`
		DataVersion int             `json:"data_version"`
		Data        json.RawMessage `json:"data"`
	}

	var scan scanEnvelope
	if err := json.Unmarshal(msg.Data, &scan); err != nil {
		panic(err)
	}

	log.Printf("Received scan: %s:%d/%s (timestamp: %d)",
		scan.Ip, scan.Port, scan.Service, scan.Timestamp)

	var payload struct {
		ResponseBytesUtf8 []byte `json:"response_bytes_utf8"`
		ResponseStr       string `json:"response_str"`
	}
	if err := json.Unmarshal(scan.Data, &payload); err != nil {
		log.Printf("failed to decode payload: %v", err)
		msg.Nack()
		return
	}

	var responseStr string
	switch scan.DataVersion {
	case scanning.V1:
		responseStr = string(payload.ResponseBytesUtf8)
	case scanning.V2:
		responseStr = payload.ResponseStr
	default:
		panic(fmt.Errorf("unknown data version: %d", scan.DataVersion))
	}
	if responseStr == "" {
		log.Print("payload missing response data")
		msg.Nack()
		return
	}

	record := storage.ScanRecord{
		Ip:          scan.Ip,
		Port:        scan.Port,
		Service:     scan.Service,
		Timestamp:   scan.Timestamp,
		ResponseStr: responseStr,
	}

	if err := store.StoreScanRecord(ctx, record); err != nil {
		log.Printf("failed storing scan: %v", err)
		msg.Nack()
		return
	}

	msg.Ack()
}

func main() {
	projectId := flag.String("project", "test-project", "GCP Project ID")
	subscriptionId := flag.String("subscription", "scan-sub", "PubSub Subscription ID")
	dbURI := flag.String("db", "postgres://postgres:postgres@localhost:5432/scans?sslmode=disable", "PostgreSQL connection URI")

	ctx := context.Background()

	flag.Parse()

	if env := os.Getenv("DATABASE_URL"); env != "" {
		*dbURI = env
	}

	db, err := sql.Open("pgx", *dbURI)
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Minute)

	defer db.Close()

	store := storage.NewPostgresStore(db)

	client, err := pubsub.NewClient(ctx, *projectId)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	sub := client.Subscription(*subscriptionId)

	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		processMessage(ctx, msg, store)
	})

	if err != nil {
		panic(err)
	}
}
