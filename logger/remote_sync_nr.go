// sad-go-logger/logger/remotesync_newrelic.go

package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// NewRelicRemoteSyncWriter implements a writer that sends log entries to New Relic Logs API.
type NewRelicRemoteSyncWriter struct {
	apiKey    string
	endpoint  string
	client    *http.Client
	buffer    []map[string]interface{}
	batchSize int
	mu        sync.Mutex
}

// NewNewRelicRemoteSyncWriter creates and returns a new NewRelicRemoteSyncWriter.
func NewNewRelicRemoteSyncWriter() RemoteSyncWriter {
	apiKey := os.Getenv("NEW_RELIC_API_KEY")
	if apiKey == "" {
		fmt.Println("NEW_RELIC_API_KEY not set. New Relic logging disabled.")
		return nil
	}

	endpoint := os.Getenv("NEW_RELIC_LOGS_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://log-api.newrelic.com/log/v1" // Default endpoint
	}

	return &NewRelicRemoteSyncWriter{
		apiKey:    apiKey,
		endpoint:  endpoint,
		client:    &http.Client{Timeout: 10 * time.Second},
		buffer:    make([]map[string]interface{}, 0, 100),
		batchSize: 100, // Can be made configurable
	}
}

func (w *NewRelicRemoteSyncWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var logEntry map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(p))
	if err := decoder.Decode(&logEntry); err != nil {
		return 0, fmt.Errorf("failed to decode log entry: %v", err)
	}

	w.buffer = append(w.buffer, logEntry)

	if len(w.buffer) >= w.batchSize {
		if err := w.flush(); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}

func (w *NewRelicRemoteSyncWriter) flush() error {
	if len(w.buffer) == 0 {
		return nil
	}

	payload := map[string]interface{}{
		"logs": w.buffer,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal log entries: %v", err)
	}

	req, err := http.NewRequest("POST", w.endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", w.apiKey)

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send logs to New Relic: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("new relic API returned unexpected status code: %d", resp.StatusCode)
	}

	w.buffer = w.buffer[:0] // Clear the buffer after successful send
	return nil
}

func (w *NewRelicRemoteSyncWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.flush()
}
