package models

import "time"

type DeviceToken struct {
	ID string
	UserID string
	APNSToken string
	UpdatedAt time.Time
}

// PushNotification описывает payload, который notification-service готовит
// к отправке. Title/Body формируется в worker'е/scheduler'е в зависимости
// от типа события.
type PushNotification struct {
	Title string
	Body string
	// Extra попадёт в APNS custom payload.
	Extra map[string]string
}
