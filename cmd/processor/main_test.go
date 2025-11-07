package main

import (
	"context"
	"encoding/json"
	"testing"

	"cloud.google.com/go/pubsub"

	"github.com/censys/scan-takehome/pkg/scanning"
	"github.com/censys/scan-takehome/pkg/storage"
)

type storeStub struct {
	last   storage.ScanRecord
	called bool
}

func (s *storeStub) StoreScanRecord(ctx context.Context, rec storage.ScanRecord) error {
	s.called = true
	s.last = rec
	return nil
}

type testEnvelope struct {
	Ip          string          `json:"ip"`
	Port        uint32          `json:"port"`
	Service     string          `json:"service"`
	Timestamp   int64           `json:"timestamp"`
	DataVersion int             `json:"data_version"`
	Data        json.RawMessage `json:"data"`
}

func newMessage(t *testing.T, version int, data any) *pubsub.Message {
	t.Helper()

	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	raw, err := json.Marshal(testEnvelope{
		Ip:          "1.2.3.4",
		Port:        443,
		Service:     "HTTPS",
		Timestamp:   123,
		DataVersion: version,
		Data:        payload,
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	return &pubsub.Message{Data: raw}
}

func TestProcessMessageV1(t *testing.T) {
	store := &storeStub{}

	msg := newMessage(t, scanning.V1, struct {
		ResponseBytesUtf8 []byte `json:"response_bytes_utf8"`
	}{
		ResponseBytesUtf8: []byte("hello"),
	})

	processMessage(context.Background(), msg, store)

	if !store.called {
		t.Fatalf("StoreScanRecord was not called")
	}

	got := store.last.ResponseStr
	want := "hello"
	if got != want {
		t.Fatalf("ResponseStr = %q, want %q", got, want)
	}
}

func TestProcessMessageV2(t *testing.T) {
	store := &storeStub{}

	msg := newMessage(t, scanning.V2, struct {
		ResponseStr string `json:"response_str"`
	}{
		ResponseStr: "hi there",
	})

	processMessage(context.Background(), msg, store)

	if !store.called {
		t.Fatalf("StoreScanRecord was not called")
	}

	got := store.last.ResponseStr
	want := "hi there"
	if got != want {
		t.Fatalf("ResponseStr = %q, want %q", got, want)
	}
}
