package main

import (
	"fmt"
	"os"

	"gopkg.in/urfave/cli.v2"
)

func main() {
	tool := &Tool{}

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   3000,
			},
			&cli.StringFlag{
				Name:    "data",
				Aliases: []string{"d"},
				Value:   ".",
			},
		},
		Action: tool.CreateServer,
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error caught: %s\n", err.Error())
	}
}
