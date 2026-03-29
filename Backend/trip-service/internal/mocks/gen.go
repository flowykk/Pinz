// Package mocks — gomock для интерфейсов репозиториев trip-service.
//
//go:generate go run go.uber.org/mock/mockgen@latest -destination=trip_repository_mock.go -package=mocks pinz/backend/trip-service/internal/repositories TripRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=trip_participant_repository_mock.go -package=mocks pinz/backend/trip-service/internal/repositories TripParticipantRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=invitation_link_repository_mock.go -package=mocks pinz/backend/trip-service/internal/repositories InvitationLinkRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=trip_settings_repository_mock.go -package=mocks pinz/backend/trip-service/internal/repositories TripSettingsRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=trip_event_publisher_mock.go -package=mocks pinz/backend/trip-service/internal/repositories TripEventPublisher
//go:generate go run go.uber.org/mock/mockgen@latest -destination=media_repository_mock.go -package=mocks pinz/backend/trip-service/internal/repositories MediaRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=pin_repository_mock.go -package=mocks pinz/backend/trip-service/internal/repositories PinRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=tag_repository_mock.go -package=mocks pinz/backend/trip-service/internal/repositories TagRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=social_repository_mock.go -package=mocks pinz/backend/trip-service/internal/repositories SocialRepositoryInterface
//go:generate go run go.uber.org/mock/mockgen@latest -destination=favourite_repository_mock.go -package=mocks pinz/backend/trip-service/internal/repositories FavouriteRepositoryInterface
package mocks
