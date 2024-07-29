// sad-go-logger/logger/remote_sync.go

package logger

type RemoteSyncWriter interface {
	Write(p []byte) (n int, err error)
	Sync() error
}
