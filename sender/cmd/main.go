package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/tousart/sender/config"
	senderAPI "github.com/tousart/sender/grpc/api"
	"github.com/tousart/sender/usecase/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Config
	cfg, err := config.MustLoad()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Usecase
	pubSubService := service.NewPubSub()
	pubSubService.StartMessageSender()

	// Server API
	serverAPI := senderAPI.CreateServerAPI(pubSubService)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
	if err != nil {
		log.Fatalf("failed to listen %d: %v", cfg.GRPC.Port, err)
	}

	server := grpc.NewServer()
	log.Println("grpc server has been created")

	senderAPI.Register(server, serverAPI)
	log.Println("sender server has been registered")

	reflection.Register(server)

	go func() {
		log.Printf("server %d has been started\n", cfg.GRPC.Port)
		if err := server.Serve(listener); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-signalChan

	pubSubService.PubSubShutdown()
	log.Println("graceful shutdown")
}
