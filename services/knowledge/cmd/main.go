package main

import (
	"log"
	"os"
	"strconv"

	"github.com/swiftbit/know-agent/pkg/infra"
	"github.com/swiftbit/know-agent/services/knowledge/internal/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/gateway"
)

func main() {
	c := config.MustLoad()
	applyEnv(c)
	if c.Grpc.KnowledgeRpc.ListenOn == "" {
		c.Grpc.KnowledgeRpc.ListenOn = ":50052"
	}

	documentConn, err := infra.DialGrpc(c.Grpc.DocumentAddr)
	if err != nil {
		log.Fatalf("dial document service: %v", err)
	}
	documentClient := gateway.NewKnowledgeDocumentClient(documentConn)

	knowledgeApp := Build(c, documentClient)

	log.Printf("knowledge service starting, http :%d", c.Http.Port)
	knowledgeApp.Start()
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
		c.Grpc.KnowledgeRpc.ListenOn = v
	}
}
