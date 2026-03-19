// Package mocks — gomock для клиентов и фасада auth.
//
//go:generate go run go.uber.org/mock/mockgen@latest -destination=auth_client_mock.go -package=mocks pinz/backend/api-gateway-service/internal/services AuthClient
//go:generate go run go.uber.org/mock/mockgen@latest -destination=auth_service_interface_mock.go -package=mocks pinz/backend/api-gateway-service/internal/services AuthServiceInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=trip_client_mock.go -package=mocks pinz/backend/api-gateway-service/internal/handlers TripClient
package mocks
