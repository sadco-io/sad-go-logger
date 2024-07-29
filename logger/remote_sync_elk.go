// sad-go-logger/logger/remotesync_elk.go

// Package logger provides functionality for logging to various outputs,
// including remote ELK (Elasticsearch, Logstash, Kibana) stacks.
package logger

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// ELKRemoteSyncWriter implements a writer that sends log entries to a remote
// Logstash instance. It supports batching, buffering, automatic reconnection,
// and optional TLS encryption.
type ELKRemoteSyncWriter struct {
	// host is the hostname or IP address of the Logstash server.
	host string

	// port is the port number on which the Logstash server is listening.
	port string

	// useTLS indicates whether to use TLS encryption for the connection.
	// If true, the connection will be established using TLS.
	useTLS bool

	// conn is the network connection to the Logstash server.
	// It may be nil if the connection is not currently established.
	conn net.Conn

	// encoder is used to JSON-encode log entries before sending them to Logstash.
	// It is initialized when a connection is established.
	encoder *json.Encoder

	// mu is used to ensure thread-safety when accessing shared resources.
	mu sync.Mutex

	// buffer holds log entries that haven't been sent to Logstash yet.
	// This allows for batching of log entries and helps handle temporary connection issues.
	buffer []map[string]interface{}

	// batchSize is the number of log entries to accumulate before sending them to Logstash.
	// When the buffer reaches this size, it will be flushed to Logstash.
	batchSize int

	// reconnectInterval is the duration to wait between connection attempts
	// when the connection to Logstash is lost.
	reconnectInterval time.Duration
}

// NewRemoteSyncWriter creates and returns a new ELKRemoteSyncWriter.
// It reads configuration from environment variables:
//   - LOGSTASH_HOST: The hostname of the Logstash server
//   - LOGSTASH_PORT: The port number of the Logstash server
//   - LOGSTASH_USE_TLS: Set to "true" to enable TLS encryption
//
// If LOGSTASH_HOST or LOGSTASH_PORT are not set, it returns nil.
func NewRemoteSyncWriter() RemoteSyncWriter {
	host := os.Getenv("LOGSTASH_HOST")
	port := os.Getenv("LOGSTASH_PORT")
	useTLS := os.Getenv("LOGSTASH_USE_TLS") == "true"
	batchSize := 100                     // Default batch size, can be made configurable
	reconnectInterval := 5 * time.Second // Default reconnect interval, can be made configurable

	if host == "" || port == "" {
		fmt.Println("LOGSTASH_HOST or LOGSTASH_PORT not set. Remote sync disabled.")
		return nil
	}

	writer := &ELKRemoteSyncWriter{
		host:              host,
		port:              port,
		useTLS:            useTLS,
		buffer:            make([]map[string]interface{}, 0, batchSize),
		batchSize:         batchSize,
		reconnectInterval: reconnectInterval,
	}

	if err := writer.connect(); err != nil {
		fmt.Printf("Failed to connect to Logstash: %v. Will retry later.\n", err)
	}

	go writer.reconnectionLoop()

	return writer
}

// connect establishes a connection to the Logstash server.
// It uses TLS if configured to do so.
func (w *ELKRemoteSyncWriter) connect() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		w.conn.Close()
	}

	var conn net.Conn
	var err error

	address := fmt.Sprintf("%s:%s", w.host, w.port)

	if w.useTLS {
		config := &tls.Config{
			InsecureSkipVerify: true, // Note: This should be configurable in production
		}
		conn, err = tls.Dial("tcp", address, config)
	} else {
		conn, err = net.Dial("tcp", address)
	}

	if err != nil {
		return err
	}

	w.conn = conn
	w.encoder = json.NewEncoder(conn)
	return nil
}

// reconnectionLoop continuously attempts to reconnect to Logstash
// if the connection is lost.
func (w *ELKRemoteSyncWriter) reconnectionLoop() {
	for {
		time.Sleep(w.reconnectInterval)
		if w.conn == nil {
			if err := w.connect(); err != nil {
				fmt.Printf("Failed to reconnect to Logstash: %v. Will retry later.\n", err)
			} else {
				fmt.Println("Successfully reconnected to Logstash.")
				w.flushBuffer()
			}
		}
	}
}

// Write implements the io.Writer interface.
// It adds the log entry to the buffer and flushes if the batch size is reached.
func (w *ELKRemoteSyncWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var logEntry map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(p))
	if err := decoder.Decode(&logEntry); err != nil {
		return 0, fmt.Errorf("failed to decode log entry: %v", err)
	}

	// Add additional fields for ELK
	logEntry["@timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	logEntry["@version"] = "1"

	w.buffer = append(w.buffer, logEntry)

	if len(w.buffer) >= w.batchSize {
		w.flushBuffer()
	}

	return len(p), nil
}

// flushBuffer sends all buffered log entries to Logstash.
// If the connection is not available, it keeps the entries in the buffer.
func (w *ELKRemoteSyncWriter) flushBuffer() {
	if w.conn == nil {
		return // Connection is not available, keep buffering
	}

	for _, entry := range w.buffer {
		if err := w.encoder.Encode(entry); err != nil {
			fmt.Printf("Failed to encode log entry for ELK: %v\n", err)
			w.conn = nil // Mark connection as failed
			return
		}
	}

	w.buffer = w.buffer[:0] // Clear the buffer
}

// Sync implements the zapcore.WriteSyncer interface.
// It flushes the buffer to ensure all logs are sent.
func (w *ELKRemoteSyncWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.flushBuffer()
	return nil
}

// Close flushes any remaining logs and closes the connection to Logstash.
func (w *ELKRemoteSyncWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.flushBuffer() // Attempt to flush any remaining logs

	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}
