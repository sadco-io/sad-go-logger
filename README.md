# SAD Go Logger

SAD Go Logger is a flexible, high-performance logging package for Go applications. It provides multi-destination logging capabilities, including console output, file writing, and remote syncing to ELK (Elasticsearch, Logstash, Kibana) stack and New Relic.

## Features

- Built on the high-performance `zap` logging library
- Multiple output destinations: console, file, ELK stack, and New Relic
- Configurable log levels
- Structured logging support
- Automatic log directory creation
- Custom time formatting
- Buffering and automatic reconnection for remote logging

## Installation

```bash
go get github.com/sadco-io/sad-go-logger
```

## Usage

Import the package in your Go code:

```go
import "github.com/sadco-io/sad-go-logger/logger"
```

Use the global `Log` variable to log messages:

```go
logger.Log.Info("Application started")
logger.Log.Error("An error occurred", zap.Error(err))
```

Use `WithFields` for structured logging:

```go
logger.WithFields(zap.String("user", "john")).Info("User logged in")
```

## Configuration

The logger is configured using environment variables. Here's a list of available options:

### General Configuration

- `SERVICE_NAME`: Name of your service (default: "sad_service")
- `LOG_LEVEL`: Logging level (default: "debug")
  - Valid options: "debug", "info", "warn", "error", "fatal", "panic"

### Remote Sync Configuration

#### ELK Stack

- `ENABLE_REMOTE_SYNC_ELK`: Set to "true" to enable ELK remote sync
- `LOGSTASH_HOST`: Hostname of your Logstash server
- `LOGSTASH_PORT`: Port number of your Logstash server
- `LOGSTASH_USE_TLS`: Set to "true" to enable TLS encryption for Logstash connection

#### New Relic

- `ENABLE_REMOTE_SYNC_NEWRELIC`: Set to "true" to enable New Relic remote sync
- `NEW_RELIC_API_KEY`: Your New Relic API key
- `NEW_RELIC_LOGS_ENDPOINT`: New Relic Logs API endpoint (optional, default: "https://log-api.newrelic.com/log/v1")

## Example Configuration

Here's an example of how to configure the logger with both ELK and New Relic enabled:

```bash
export SERVICE_NAME="my-awesome-service"
export LOG_LEVEL="info"

# ELK Configuration
export ENABLE_REMOTE_SYNC_ELK="true"
export LOGSTASH_HOST="logstash.example.com"
export LOGSTASH_PORT="5000"
export LOGSTASH_USE_TLS="true"

# New Relic Configuration
export ENABLE_REMOTE_SYNC_NEWRELIC="true"
export NEW_RELIC_API_KEY="your-new-relic-api-key-here"
```

## Log File Locations

Log files are automatically created in the `./logs` directory:

- `./logs/logs.txt`: Contains all log entries
- `./logs/errors.txt`: Contains only error-level and above log entries

## Performance Considerations

- The logger uses buffering for remote syncing to minimize performance impact.
- Logs are sent to remote destinations in batches to reduce network overhead.
- If a remote destination is unavailable, logs are buffered in memory and the logger will attempt to reconnect periodically.

## Thread Safety

The logger is designed to be thread-safe and can be safely used from multiple goroutines concurrently.

## Extending the Logger

The logger uses the `RemoteSyncWriter` interface for remote logging implementations. You can create new implementations of this interface to add support for additional remote logging services.

## Contributing

Contributions to SAD Go Logger are welcome! Please submit pull requests with any enhancements, bug fixes, or new features.

## License

For private use within sadco-io services for now.

## Support

For issues, feature requests, or questions, please file an issue on the GitHub repository.
