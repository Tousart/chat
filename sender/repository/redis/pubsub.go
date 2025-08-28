package redis

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
	senderv1 "github.com/tousart/protochat/gen/go/sender"
)

type PubSubRepo struct {
	redisClient     *redis.Client
	subscriber      *redis.PubSub
	chatSubscribers map[string]map[string]senderv1.Sender_SendMessageServer
}

func NewPubSubRepo(addr string) *PubSubRepo {
	rc := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	s := rc.Subscribe(context.Background())

	cs := make(map[string]map[string]senderv1.Sender_SendMessageServer)

	return &PubSubRepo{
		redisClient:     rc,
		subscriber:      s,
		chatSubscribers: cs,
	}
}

func (ps *PubSubRepo) PubSubShutdown() {
	ps.subscriber.Close()
	ps.redisClient.Close()
}

func (ps *PubSubRepo) SubscribeToChat(stream senderv1.Sender_SendMessageServer, chatID, userID string) error {
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

func (ps *PubSubRepo) UnsubscribeFromChat(chatID, userID string) {
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

func (ps *PubSubRepo) PublishMessage(chatID string, payload []byte) error {
	if err := ps.redisClient.Publish(context.Background(), chatID, payload).Err(); err != nil {
		log.Printf("failed to publish message: %v\n", err)
		return err
	}
	return nil
}

func (ps *PubSubRepo) GetMessagesFromChannel() <-chan *redis.Message {
	return ps.subscriber.Channel()
}

func (ps *PubSubRepo) GetUsersStreamsFromChatID(chatID string) map[string]senderv1.Sender_SendMessageServer {
	return ps.chatSubscribers[chatID]
}
