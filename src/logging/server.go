package logging

import (
	"bytes"
	"container/ring"
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
)

// Size of buffer for incoming log channel.
const incomingChannelBufferSize = 5

// Number of log entries to keep in circular buffer
const bufferSize = 100

// LogServer implements logrus.Hook and http.Handler interfaces
type LogServer struct {
	incoming chan *logrus.Entry

	buffer *ring.Ring
	mutex  *sync.RWMutex
}

// NewLogServer returns a new LogServer
func NewLogServer() *LogServer {
	logServer := LogServer{}

	logServer.incoming = make(chan *logrus.Entry, incomingChannelBufferSize)

	// set up log buffer and RWMutex
	logServer.buffer = ring.New(bufferSize)
	logServer.mutex = &sync.RWMutex{}

	// start a goroutine handling incoming log entries
	go func() {
		for entry := range logServer.incoming {
			logServer.mutex.Lock()
			logServer.buffer.Value = entry
			// Point to next value. For readers the buffer always points to the oldest log entry.
			logServer.buffer = logServer.buffer.Next()
			logServer.mutex.Unlock()
		}
	}()

	return &logServer
}

// Levels implements the logrus.Hook interface
func (logServer *LogServer) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
	}
}

// Fire implements the logrus.Hook interface
func (logServer *LogServer) Fire(entry *logrus.Entry) error {
	select {
	case logServer.incoming <- entry:
		return nil
	default:
		return errors.New("LogServer not accepting entries into buffer, dropping entry.")
	}
}

// Use UTC in as timestamp (from https://stackoverflow.com/a/40502637)
type UTCFormatter struct {
	logrus.Formatter
}

func (u UTCFormatter) Format(e *logrus.Entry) ([]byte, error) {
	e.Time = e.Time.UTC()
	return u.Formatter.Format(e)
}

var formatter = UTCFormatter{&logrus.JSONFormatter{}}

// Implement net/http Handler interface
func (logServer *LogServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logServer.mutex.RLock()
	defer logServer.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json; charset=utf-8") // normal header
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// first collect entries in slice so that we can intersperse with ",". See also: https://www.happyassassin.net/2017/09/07/a-modest-proposal/
	entries := make([][]byte, 0, bufferSize)

	logServer.buffer.Do(func(i interface{}) {
		entry, ok := i.(*logrus.Entry)
		if !ok {
			return
		}

		encoded, encodeErr := formatter.Format(entry)
		if encodeErr != nil {
			return
		}
		entries = append(entries, encoded)
	})

	io.WriteString(w, "[")
	w.Write(bytes.Join(entries, []byte(",")))
	io.WriteString(w, "]")

}
