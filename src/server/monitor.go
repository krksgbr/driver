package server

import (
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

func startMonitor(log *logrus.Entry) {
	var m runtime.MemStats

	c := time.NewTicker(30 * time.Second).C

	for range c {
		runtime.ReadMemStats(&m)
		log.WithField("heapAlloc", m.HeapAlloc).WithField("routines", runtime.NumGoroutine()).Info("Monitoring runtime.")
	}
}
