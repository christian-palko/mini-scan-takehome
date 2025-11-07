# Mini-Scan

This repository contains two services:

- **Scanner** – publishes random scan messages to a Pub/Sub topic.
- **Processor** – reads those messages, keeps the newest response for each `(ip, port, service)`, and stores it in Postgres.

## Features

- **At-least-once processing:** the processor only acks a message after a successful database write. Failures result in `Nack` and redelivery.
- **Latest-record wins:** messages with an older timestamp than what’s stored are skipped.
- **Pluggable storage:** the processor depends on the `storage.Store` interface, so swapping Postgres for another store only requires a new implementation.
- **Integration + unit tests:** one test runs against Postgres to check insert/update/stale handling, and another covers v1/v2 payload parsing.

## Requirements

- Docker Desktop
- Go 1.20 or newer

## Setup

1. Clone this repo:
   ```bash
   git clone https://github.com/christian-palko/mini-scan-takehome.git
   cd mini-scan-takehome
   ```
2. Ensure Docker Desktop is running

## Development Workflow

```bash
make up
```

This builds and starts the Pub/Sub emulator, Postgres, scanner, and processor, waits for them to become healthy, applies the initial migration, and lastly tails the processor/scanner logs. With `air` running, edits to Go files trigger a rebuild automatically.

Once this is running, the provided scanner will begin sending Pub/Sub messages of random scans to the processor, and the processor will continually upsert only fresh records.

### Stopping

```bash
make down
```

## Testing

```bash
make test
```

This runs `go test ./...`. Highlights:

- `pkg/storage/storage_test.go` spins up a temporary Postgres container, applies `db/migrations/0001_init.sql`, and verifies insert/ignore/update behavior.
- `cmd/processor/main_test.go` exercises the message handler for both v1 and v2 payloads with a stubbed store.

## Project Structure

- `cmd/processor`: Pub/Sub consumer that writes to Postgres.
- `cmd/scanner`: Sample publisher feeding random scans.
- `pkg/storage`: Database models and the staleless `StoreScanRecord` logic.
- `storage.Store`: simple interface so a different store (e.g., DynamoDB, SQLite) can be plugged in without touching the processor.
- `db/migrations`: SQL schema

## Future Improvements

- Switch to a formal migration tool (e.g., `golang-migrate`)
- Add metrics / tracing
- Build CI pipeline that runs `make test` and lints the code.
