package api

import (
	senderv1 "github.com/tousart/protochat/gen/go/sender"
	"github.com/tousart/sender/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type serverAPI struct {
	senderv1.UnimplementedSenderServer
	usecase usecase.PubSub
}

func CreateServerAPI(uc usecase.PubSub) *serverAPI {
	return &serverAPI{usecase: uc}
}

func Register(gRPCServer *grpc.Server, api *serverAPI) {
	senderv1.RegisterSenderServer(gRPCServer, api)
}

func (s *serverAPI) SendMessage(stream senderv1.Sender_SendMessageServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Error(codes.InvalidArgument, "metadata is required")
	}

	chatID := md.Get("chat_id")[0]
	userID := md.Get("user_id")[0]

	if chatID == "" || userID == "" {
		return status.Error(codes.InvalidArgument, "metadata is empty")
	}

	// Add clients stream to online users map
	err := s.usecase.SubscribeToChat(stream, chatID, userID)
	if err != nil {
		return status.Error(codes.Internal, "failed to add clients stream")
	}

	// Start publish messages from users stream
	pubErrChan := make(chan error)
	s.usecase.PublishMessages(stream, chatID, pubErrChan)

	err = <-pubErrChan
	s.usecase.UnsubscribeFromChat(chatID, userID)

	return err
}
