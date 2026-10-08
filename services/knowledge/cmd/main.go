package main

import (
	"log"

	"github.com/swiftbit/know-agent/services/knowledge/internal/config"
)

func main() {
	c := config.MustLoad()
	if c.Rpc.ListenOn == "" {
		c.Rpc.ListenOn = ":50052"
	}

	server, err := Build(c)
	if err != nil {
		log.Fatalf("build knowledge service: %v", err)
	}

	log.Printf("knowledge service starting, http :%d", c.Http.Port)
	server.Start()
}
