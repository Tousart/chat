package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tousart/sender/config"
	senderAPI "github.com/tousart/sender/grpc/api"
	pkgGRPC "github.com/tousart/sender/pkg/grpc"
	redisRepo "github.com/tousart/sender/repository/redis"
	"github.com/tousart/sender/usecase/service"
)

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Config
	cfg, err := config.MustLoad()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Repository
	pubSubRepo := redisRepo.NewPubSubRepo(cfg.Redis.Address)

	// Usecase
	pubSubService := service.NewPubSub(pubSubRepo)

	// Server API
	serverAPI := senderAPI.CreateServerAPI(pubSubService)
	serverAPI.StartMessageSender()

	// Run
	errChan := make(chan error)
	pkgGRPC.CreateAndRunServer(serverAPI, pubSubService, cfg.GRPC.Port, errChan)

	select {
	case <-signalChan:
		pubSubService.PubSubShutdown()
		log.Println("graceful shutdown")
	case err := <-errChan:
		log.Fatalf("run server error: %v", err)
	}
}
