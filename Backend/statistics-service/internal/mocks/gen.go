// Package mocks — gomock для интерфейсов репозиториев statistics-service.
//
//go:generate go run go.uber.org/mock/mockgen@latest -destination=user_stats_repository_mock.go -package=mocks pinz/backend/statistics-service/internal/repositories UserStatsRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=trip_locations_repository_mock.go -package=mocks pinz/backend/statistics-service/internal/repositories TripLocationsRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=geo_registry_repository_mock.go -package=mocks pinz/backend/statistics-service/internal/repositories GeoRegistryRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=event_log_repository_mock.go -package=mocks pinz/backend/statistics-service/internal/repositories EventLogRepositoryInterface
package mocks
