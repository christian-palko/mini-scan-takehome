CREATE TABLE IF NOT EXISTS scan_records (
    ip TEXT NOT NULL,
    port INTEGER NOT NULL,
    service TEXT NOT NULL,
    timestamp BIGINT NOT NULL,
    response_str TEXT NOT NULL,
    PRIMARY KEY (ip, port, service)
);

