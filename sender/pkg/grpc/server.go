package grpc

import (
	"fmt"
	"log"
	"net"

	senderAPI "github.com/tousart/sender/grpc/api"
	"github.com/tousart/sender/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func CreateAndRunServer(pubSubService usecase.PubSub, port int, errChan chan error) {
	// Server API
	serverAPI := senderAPI.CreateServerAPI(pubSubService)

	// Creating server
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		errChan <- fmt.Errorf("failed to listen %d: %v", port, err)
		return
	}

	server := grpc.NewServer()
	log.Println("grpc server has been created")

	senderAPI.Register(server, serverAPI)
	log.Println("sender server has been registered")

	reflection.Register(server)

	go func() {
		log.Printf("server %d has been started\n", port)
		if err := server.Serve(listener); err != nil {
			errChan <- fmt.Errorf("failed to serve: %v", err)
			return
		}
	}()
}
