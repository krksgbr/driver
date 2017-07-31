package main

import (
	"context"
	"os"

	"github.com/dividat/driver-go/mock_senso"
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
		{
			Name:  "mock",
			Usage: "mock a Dividat Senso",
			Action: func(c *cli.Context) error {

				ctx := context.Background()

				mSenso := mock_senso.Default()
				mSenso.Start(ctx)

				return nil
			},
		},
	}

	app.Run(os.Args)
}
