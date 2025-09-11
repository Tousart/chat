package api

import (
	"encoding/json"
	"log"
	"sync"

	senderv1 "github.com/tousart/protochat/gen/go/sender"
	"github.com/tousart/sender/models"
	"github.com/tousart/sender/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ServerAPI struct {
	senderv1.UnimplementedSenderServer
	usecase usecase.PubSub
	mu      *sync.RWMutex
}

func CreateServerAPI(uc usecase.PubSub) *ServerAPI {
	mu := new(sync.RWMutex)
	return &ServerAPI{
		usecase: uc,
		mu:      mu,
	}
}

func Register(gRPCServer *grpc.Server, api *ServerAPI) {
	senderv1.RegisterSenderServer(gRPCServer, api)
}

func (s *ServerAPI) SendMessage(stream senderv1.Sender_SendMessageServer) error {
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
	s.mu.Lock()
	err := s.usecase.SubscribeToChat(stream, chatID, userID)
	if err != nil {
		return status.Error(codes.Internal, "failed to add clients stream")
	}
	s.mu.Unlock()

	// Start publish messages from users stream
	pubErrChan := make(chan error)
	s.PublishMessages(stream, chatID, pubErrChan)

	err = <-pubErrChan

	s.mu.Lock()
	s.usecase.UnsubscribeFromChat(chatID, userID)
	s.mu.Unlock()

	return err
}

func (s *ServerAPI) PublishMessages(stream senderv1.Sender_SendMessageServer, chatID string, pubErrChan chan error) {

	go func() {
		for {
			in, err := stream.Recv()
			if err != nil {
				pubErrChan <- err
				return
			}

			if in.UserId == "" {
				pubErrChan <- status.Error(codes.InvalidArgument, "user id is required")
				return
			}

			if in.ChatId < 0 {
				pubErrChan <- status.Error(codes.InvalidArgument, "incorrect chat id")
				return
			}

			if in.Text == "" {
				pubErrChan <- status.Error(codes.InvalidArgument, "message body is required")
				return
			}

			user := &models.User{
				UserID: in.UserId,
				ChatID: in.ChatId,
				Text:   in.Text,
			}

			payload, err := json.Marshal(user)
			if err != nil {
				pubErrChan <- err
				return
			}

			if err := s.usecase.PublishMessages(chatID, payload); err != nil {
				pubErrChan <- err
				return
			}

		}
	}()

}

// General sender! Up in main
func (s *ServerAPI) StartMessageSender() {

	go func() {
		user := models.User{}
		for message := range s.usecase.GetMessagesFromChannel() {
			if err := json.Unmarshal([]byte(message.Payload), &user); err != nil {
				log.Printf("failed to unmarshal messages payload: %v\n", err)
				continue
			}

			s.mu.RLock()
			chatID := message.Channel
			for _, stream := range s.usecase.GetUsersStreamsFromChatID(chatID) {
				err := stream.Send(&senderv1.SendMessageResponse{
					Response: &senderv1.SendMessageResponse_Message{
						Message: &senderv1.ChatMessage{
							UserId: user.UserID,
							ChatId: user.ChatID,
							Text:   user.Text,
						},
					},
				})
				if err != nil {
					log.Printf("failed to send message to %s, from chat: %d, error: %v\n", user.UserID, user.ChatID, err)
					// обработка ошибки
				}
			}
			s.mu.RUnlock()

		}
	}()

}
