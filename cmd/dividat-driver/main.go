package main

import (
	"os"

	"github.com/dividat/driver-go/server"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	app.Name = "dividat-driver"
	app.Usage = "Dividat hardware drivers"

	app.Commands = []cli.Command{
		{
			Name:  "start",
			Usage: "start the driver",
			Action: func(c *cli.Context) error {
				server.Start()
				return nil
			},
		},
	}

	app.Run(os.Args)
}
