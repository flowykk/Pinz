// Package mocks — gomock для интерфейсов notification-service.
//
//go:generate go run go.uber.org/mock/mockgen@latest -destination=device_tokens_repository_mock.go -package=mocks pinz/backend/notification-service/internal/repositories DeviceTokensRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=notification_log_repository_mock.go -package=mocks pinz/backend/notification-service/internal/repositories NotificationLogRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=trip_client_mock.go -package=mocks pinz/backend/notification-service/internal/repositories TripClientInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=apns_sender_mock.go -package=mocks pinz/backend/notification-service/internal/apns Sender
package mocks
