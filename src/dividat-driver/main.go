package main

// Start up driver as a service

// TODO: implement logging with service.Logger

import (
	"context"
	"log"

	"dividat-driver/server"

	"github.com/kardianos/service"
)

type program struct {
	close context.CancelFunc
}

func (p *program) Start(s service.Service) error {
	p.close = server.Start(service.Interactive())
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
