package repository

import (
	"github.com/redis/go-redis/v9"
	senderv1 "github.com/tousart/protochat/gen/go/sender"
)

type PubSubRepo interface {
	PubSubShutdown()
	SubscribeToChat(stream senderv1.Sender_SendMessageServer, chatID, userID string) error
	UnsubscribeFromChat(chatID, userID string)
	PublishMessage(chatID string, payload []byte) error
	GetMessagesFromChannel() <-chan *redis.Message
	GetUsersStreamsFromChatID(chatID string) map[string]senderv1.Sender_SendMessageServer
}
