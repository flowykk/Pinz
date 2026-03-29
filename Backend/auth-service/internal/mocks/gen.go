// Package mocks содержит автогенерируемые gomock-реализации интерфейсов сервисов.
//
//go:generate go run go.uber.org/mock/mockgen@latest -destination=user_repository_mock.go -package=mocks pinz/backend/auth-service/internal/services UserRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=redis_repository_mock.go -package=mocks pinz/backend/auth-service/internal/services RedisRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=credential_repository_mock.go -package=mocks pinz/backend/auth-service/internal/services CredentialRepositoryInterface
package mocks
