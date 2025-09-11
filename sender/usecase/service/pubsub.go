package service

import (
	"sync"

	"github.com/redis/go-redis/v9"
	senderv1 "github.com/tousart/protochat/gen/go/sender"
	"github.com/tousart/sender/repository"
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

func (ps *PubSub) PublishMessages(chatID string, payload []byte) error {
	if err := ps.pubSub.PublishMessage(chatID, payload); err != nil {
		return err
	}
	return nil
}

func (ps *PubSub) GetMessagesFromChannel() <-chan *redis.Message {
	return ps.pubSub.GetMessagesFromChannel()
}

func (ps *PubSub) GetUsersStreamsFromChatID(chatID string) map[string]senderv1.Sender_SendMessageServer {
	return ps.pubSub.GetUsersStreamsFromChatID(chatID)
}
