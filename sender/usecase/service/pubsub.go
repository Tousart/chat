package service

import (
	"encoding/json"
	"log"
	"sync"

	senderv1 "github.com/tousart/protochat/gen/go/sender"
	"github.com/tousart/sender/models"
	"github.com/tousart/sender/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PubSub struct {
	mu     *sync.RWMutex
	pubSub repository.PubSubRepo
}

func NewPubSub(pubSubRepo repository.PubSubRepo) *PubSub {
	mu := new(sync.RWMutex)

	return &PubSub{
		mu:     mu,
		pubSub: pubSubRepo,
	}
}

// Up in main with defer
func (ps *PubSub) PubSubShutdown() {
	ps.pubSub.PubSubShutdown()
}

func (ps *PubSub) SubscribeToChat(stream senderv1.Sender_SendMessageServer, chatID, userID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if err := ps.pubSub.SubscribeToChat(stream, chatID, userID); err != nil {
		return err
	}

	return nil
}

func (ps *PubSub) UnsubscribeFromChat(chatID, userID string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.pubSub.UnsubscribeFromChat(chatID, userID)
}

func (ps *PubSub) PublishMessages(stream senderv1.Sender_SendMessageServer, chatID string, pubErrChan chan error) {

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

			if err := ps.pubSub.PublishMessage(chatID, payload); err != nil {
				pubErrChan <- err
				return
			}

		}
	}()

}

// General sender! Up in main
func (ps *PubSub) StartMessageSender() {

	go func() {
		user := models.User{}
		for message := range ps.pubSub.GetMessagesFromChannel() {
			if err := json.Unmarshal([]byte(message.Payload), &user); err != nil {
				log.Printf("failed to unmarshal messages payload: %v\n", err)
				continue
			}

			// ps.mu.RLock()
			// currentStreams := []models.User{}
			chatID := message.Channel
			for userID, stream := range ps.pubSub.GetUsersStreamsFromChatID(chatID) {
				// currentStreams = append(currentStreams, )
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
					ps.UnsubscribeFromChat(chatID, userID)
				}
			}
			// ps.mu.RUnlock()

		}
	}()

}
