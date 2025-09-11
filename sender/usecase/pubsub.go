package usecase

import (
	"github.com/redis/go-redis/v9"
	senderv1 "github.com/tousart/protochat/gen/go/sender"
)

type PubSub interface {
	SubscribeToChat(stream senderv1.Sender_SendMessageServer, chatID, userID string) error
	GetUsersStreamsFromChatID(chatID string) map[string]senderv1.Sender_SendMessageServer
	PublishMessages(chatID string, payload []byte) error
	UnsubscribeFromChat(chatID, userID string)
	GetMessagesFromChannel() <-chan *redis.Message
	PubSubShutdown()
}
