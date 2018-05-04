package main

// Start up driver as a service

import (
	"context"
	"io/ioutil"
	"log"

	"dividat-driver/logging"
	"dividat-driver/server"

	"github.com/kardianos/service"
	"github.com/sirupsen/logrus"
)

type program struct {
	close context.CancelFunc
}

func (p *program) Start(s service.Service) error {
	// Set up logging
	logger := logrus.New()
	if !service.Interactive() {
		logger.Out = ioutil.Discard
		if systemLogger, err := s.SystemLogger(nil); err == nil {
			logger.AddHook(logging.NewSystemHook(systemLogger))
		}
	}
	logger.SetLevel(logrus.DebugLevel)

	p.close = server.Start(logger)
	return nil
}
func (p *program) Stop(s service.Service) error {
	p.close()
	return nil
}

func main() {
	svcConfig := &service.Config{
		Name:        "DividatDriver",
		DisplayName: "Dividat Driver",
		Description: "Dividat Driver application for hardware connectivity.",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(s.Run())
}
