package server

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/sirupsen/logrus"
)

func startMonitor(log *logrus.Entry) {
	var m runtime.MemStats
	c := make(chan os.Signal, 1)

	signal.Notify(c, syscall.SIGUSR1)

	for sig := range c {
		if sig == syscall.SIGUSR1 {
			runtime.ReadMemStats(&m)
			log.WithField("sysMem", m.Sys/1024).WithField("routines", runtime.NumGoroutine()).Info("received SIGUSR1")
		}
	}
}
