package models

type User struct {
	UserID string `json:"user_id"`
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}
