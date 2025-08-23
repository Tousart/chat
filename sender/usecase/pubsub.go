package usecase

import (
	senderv1 "github.com/tousart/protochat/gen/go/sender"
)

type PubSub interface {
	SubscribeToChat(stream senderv1.Sender_SendMessageServer, chatID, userID string) error
	PublishMessages(stream senderv1.Sender_SendMessageServer, chatID string, pubErrChan chan error)
	UnsubscribeFromChat(chatID, userID string)
	PubSubShutdown()
	StartMessageSender()
}
