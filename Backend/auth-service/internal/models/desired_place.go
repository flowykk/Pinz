package models

import "time"

// DesiredPlace — желаемое место пользователя. ImageURL хранит s3-ключ
// (пустой = картинка не загружена), наружу отдаётся presigned URL.
type DesiredPlace struct {
	ID          string
	UserID      string
	Name        string
	Description string
	ImageURL    string
	CreatedAt   time.Time
}
