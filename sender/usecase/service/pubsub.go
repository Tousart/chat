package service

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
	senderv1 "github.com/tousart/protochat/gen/go/sender"
	"github.com/tousart/sender/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PubSub struct {
	mu              *sync.RWMutex
	redisClient     *redis.Client
	subscriber      *redis.PubSub
	chatSubscribers map[string]map[string]senderv1.Sender_SendMessageServer
}

func NewPubSub() *PubSub {
	mu := new(sync.RWMutex)

	client := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	subscriber := client.Subscribe(context.Background())

	cs := make(map[string]map[string]senderv1.Sender_SendMessageServer)

	return &PubSub{
		mu:              mu,
		redisClient:     client,
		subscriber:      subscriber,
		chatSubscribers: cs,
	}
}

// Up in main with defer
func (ps *PubSub) PubSubShutdown() {
	ps.subscriber.Close()
	ps.redisClient.Close()
}

func (ps *PubSub) SubscribeToChat(stream senderv1.Sender_SendMessageServer, chatID, userID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, ok := ps.chatSubscribers[chatID]; !ok {
		ps.chatSubscribers[chatID] = make(map[string]senderv1.Sender_SendMessageServer)
	}

	ps.chatSubscribers[chatID][userID] = stream

	if len(ps.chatSubscribers[chatID]) == 1 {
		err := ps.subscriber.Subscribe(context.Background(), chatID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ps *PubSub) UnsubscribeFromChat(chatID, userID string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, ok := ps.chatSubscribers[chatID]; !ok {
		log.Printf("failed to unsubscribe: chat id %s is not exists\n", chatID)
		return
	}

	if _, ok := ps.chatSubscribers[chatID][userID]; !ok {
		log.Printf("failed to unsubscribe: user %s not in the chat %s\n", userID, chatID)
		return
	}

	delete(ps.chatSubscribers[chatID], userID)

	if len(ps.chatSubscribers[chatID]) == 0 {
		delete(ps.chatSubscribers[chatID], chatID)
		err := ps.subscriber.Unsubscribe(context.Background(), chatID)
		if err != nil {
			log.Printf("unsubscribe error: %v\n", err)
			return
		}
	}
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

			if err := ps.redisClient.Publish(context.Background(), chatID, payload).Err(); err != nil {
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
		for message := range ps.subscriber.Channel() {
			if err := json.Unmarshal([]byte(message.Payload), &user); err != nil {
				log.Printf("failed to unmarshal messages payload: %v\n", err)
				continue
			}

			// ps.mu.RLock()
			// currentStreams := []models.User{}
			chatID := message.Channel
			for userID, stream := range ps.chatSubscribers[chatID] {
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
