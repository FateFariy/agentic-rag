package main

import (
	"log"
	"os"
	"strconv"

	"github.com/swiftbit/know-agent/services/chat/internal/config"
)

func main() {
	c := config.MustLoad()
	applyEnv(c)

	chatApp, err := Build(c)
	if err != nil {
		log.Fatalf("build chat service: %v", err)
	}

	log.Printf("chat service starting, http :%d", c.Http.Port)
	chatApp.Start()
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
	if v := os.Getenv("KNOWLEDGE_GRPC_ADDR"); v != "" {
		c.Grpc.KnowledgeAddr = v
	}
}
