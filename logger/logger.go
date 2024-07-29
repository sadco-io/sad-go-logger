// sad-go-logger/logger/logger.go

package logger

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger
var hostname string
var serviceName string
var initLog map[string]interface{}

func init() {
	initLog = make(map[string]interface{})

	var err error
	hostname, err = os.Hostname()
	if err != nil {
		initLog["hostnameMessage"] = fmt.Sprintf("Error retrieving hostname: %v", err) + "Setting hostname to unkw"
		hostname = "unkw"
	}

	serviceName = os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		initLog["serviceNameMessage"] = "SERVICE_NAME is not set, using sad_service as default"
		serviceName = "sad_service"
	}

	// Create logs directory if not exists
	if _, err := os.Stat("./logs"); os.IsNotExist(err) {
		if err := os.Mkdir("./logs", 0755); err != nil {
			fmt.Printf("Warning: Unable to create log directory './logs': %v\n", err)
		}
	}

	// Open or create log files in the logs directory
	file, err := os.OpenFile("./logs/logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	errorLog, err := os.OpenFile("./logs/errors.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	logLevel := os.Getenv("LOG_LEVEL")
	var zapLevel zapcore.Level
	if logLevel == "" {
		logLevel = "debug"
	}
	switch logLevel {
	case "debug":
		zapLevel = zap.DebugLevel
	case "info":
		zapLevel = zap.InfoLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	case "fatal":
		zapLevel = zap.FatalLevel
	case "panic":
		zapLevel = zap.PanicLevel
	default:
		zapLevel = zap.DebugLevel
	}

	// Create a custom encoder config
	encoderConfig := zapcore.EncoderConfig{
		MessageKey: "message",
		LevelKey:   "level",
		TimeKey:    "datetime",
		EncodeTime: zapcore.TimeEncoder(func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
		}),
		EncodeLevel:      zapcore.CapitalLevelEncoder,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		ConsoleSeparator: ". ", // Use dot and space as the separator
	}

	// Create a custom core that writes to both stdout and file
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)

	stdoutSink := zapcore.AddSync(os.Stdout)
	fileSink := zapcore.AddSync(file)
	errorFileSink := zapcore.AddSync(errorLog)

	// Create a core for stdout and file
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, stdoutSink, zapLevel),
		zapcore.NewCore(fileEncoder, fileSink, zapLevel),
		zapcore.NewCore(fileEncoder, errorFileSink, zap.ErrorLevel),
	)

	// Check if remote sync is enabled for ELK
	if os.Getenv("ENABLE_REMOTE_SYNC_ELK") == "true" {
		remoteSyncWriter := NewRemoteSyncWriter()
		if remoteSyncWriter != nil {
			remoteSink := zapcore.AddSync(remoteSyncWriter)
			core = zapcore.NewTee(core, zapcore.NewCore(fileEncoder, remoteSink, zapLevel))
		}
	}

	// Check if remote sync is enabled for New Relic
	if os.Getenv("ENABLE_REMOTE_SYNC_NEWRELIC") == "true" {
		newRelicWriter := NewNewRelicRemoteSyncWriter()
		if newRelicWriter != nil {
			newRelicSink := zapcore.AddSync(newRelicWriter)
			core = zapcore.NewTee(core, zapcore.NewCore(fileEncoder, newRelicSink, zapLevel))
		}
	}

	// Create the logger
	Log = zap.New(core, zap.AddCaller(), zap.Fields(
		zap.String("hostname", hostname),
		zap.String("serviceName", serviceName),
	))

	Log.Debug("Logger initialized")

	if len(initLog) > 0 {
		for key, value := range initLog {
			Log.Sugar().Infof("%s, %v", key, value)
		}
	}
	Log.Info("Logger set to " + logLevel + " level")
}

// WithFields adds structured context to the logger.
func WithFields(fields ...zap.Field) *zap.Logger {
	return Log.With(fields...)
}
