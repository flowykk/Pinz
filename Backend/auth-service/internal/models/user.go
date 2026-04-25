package models

import "time"

type User struct {
	ID string
	Email string
	Username string
	AvatarURL string
	CreatedAt time.Time
}
