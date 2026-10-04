package main

import (
	"log"
	"os"
	"strconv"

	"github.com/swiftbit/know-agent/pkg/config"
	pkggrpcclient "github.com/swiftbit/know-agent/pkg/grpcclient"
	"github.com/swiftbit/know-agent/pkg/grpcserver"
	"github.com/swiftbit/know-agent/services/knowledge/internal/bootstrap"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/gateway"
)

func main() {
	c := config.MustLoad()
	applyEnv(c)

	documentConn, err := pkggrpcclient.Dial(c.Grpc.DocumentAddr)
	if err != nil {
		log.Fatalf("dial document service: %v", err)
	}
	documentClient := gateway.NewKnowledgeDocumentClient(documentConn)

	knowledgeApp := bootstrap.Build(c, documentClient)

	go func() {
		addr := c.Grpc.KnowledgeListen
		if addr == "" {
			addr = ":50052"
		}
		if err := grpcserver.RunGRPC(addr, knowledgeApp.RegisterGRPC); err != nil {
			log.Fatalf("knowledge gRPC server: %v", err)
		}
	}()

	log.Printf("knowledge service starting, http :%d", c.Http.Port)
	knowledgeApp.HTTPServer().Start()
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
		c.Grpc.KnowledgeListen = v
	}
}
