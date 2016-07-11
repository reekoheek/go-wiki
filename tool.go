package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"gopkg.in/urfave/cli.v2"
)

type Tool struct {
}

func (t *Tool) CreateServer(c *cli.Context) error {
	addr := ":" + strconv.Itoa(c.Int("port"))
	dataDir := c.String("data")

	log.Printf("DEBUG  = %s\n", os.Getenv("DEBUG"))
	log.Printf("DATA   = %s\n", dataDir)
	log.Printf("ADDR   = %s\n", addr)

	server, err := NewServer(dataDir)
	if err != nil {
		return err
	}
	http.HandleFunc("/", server.Callback())
	return http.ListenAndServe(addr, nil)
}
