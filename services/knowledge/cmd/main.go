package main

import (
	"log"
	"os"
	"strconv"

	"github.com/swiftbit/know-agent/services/knowledge/internal/config"
)

func main() {
	c := config.MustLoad()
	applyEnv(c)
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

func applyEnv(c *config.Config) {
	if p := os.Getenv("HTTP_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			c.Http.Port = v
		}
	}
	if v := os.Getenv("DOCUMENT_GRPC_ADDR"); v != "" {
		c.Grpc.DocumentAddr = v
	}
	if v := os.Getenv("KNOWLEDGE_GRPC_LISTEN"); v != "" {
		c.Rpc.ListenOn = v
	}
}
