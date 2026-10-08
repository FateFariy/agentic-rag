package main

import (
	"log"

	"github.com/swiftbit/know-agent/services/document/internal/config"
)

func main() {
	c := config.MustLoad()
	if c.Rpc.ListenOn == "" {
		c.Rpc.ListenOn = ":50051"
	}

	server, err := Build(c)
	if err != nil {
		log.Fatalf("build document service: %v", err)
	}

	log.Printf("document service starting, http :%d", c.Http.Port)
	server.Start()
}
