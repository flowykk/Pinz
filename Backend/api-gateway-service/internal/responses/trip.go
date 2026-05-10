package responses

// Trip is the REST representation of a trip.
type Trip struct {
	ID string `json:"id"`
	OwnerUserID string `json:"owner_user_id"`
	Name string `json:"name"`
	Description string `json:"description"`
	Category string `json:"category"`
	Season string `json:"season"`
	Status string `json:"status"`
	PrivacyLevel string `json:"privacy_level"`
	StartDateUnix int64 `json:"start_date_unix,omitempty"`
	EndDateUnix int64 `json:"end_date_unix,omitempty"`
	LikesCount int32 `json:"likes_count"`
	DislikesCount int32 `json:"dislikes_count"`
	MediaCount int32 `json:"media_count"`
	ParticipantsCount int32 `json:"participants_count"`
	CoverURL string `json:"cover_url,omitempty"`
	IsPublished bool `json:"is_published"`
	IsGenerated bool `json:"is_generated"`
	CreatedAtUnix int64 `json:"created_at_unix"`
	UpdatedAtUnix int64 `json:"updated_at_unix"`
	// ShareURL — universal-link на просмотр трипа (ТЗ 3.4): копирование и
	// отправка во внешние мессенджеры. Формируется gateway по env
	// TRIP_SHARE_LINK_BASE; пустой, если переменная не задана.
	ShareURL string `json:"share_url,omitempty"`
}

// GetTripResponse is the response for GET /api/v1/trips/{id} (trip with pins, participants and current user settings).
// ActiveAddMediaSession — есть активная add-media-сессия (null иначе).
// Клиент по этому полю + trip.status выбирает экран add-media флоу.
type GetTripResponse struct {
	Trip Trip `json:"trip"`
	Pins []TripPin `json:"pins"`
	Participants []TripParticipant `json:"participants"`
	CurrentUserSettings TripSettings `json:"current_user_settings"`
	ActiveAddMediaSession *ActiveAddMediaSession `json:"active_add_media_session,omitempty"`
}

// TripParticipant — участник трипа в ответе GetTrip.
// privacy_level — per-user выбор этого участника ("private" если записи нет).
// role — "admin" | "member". username/avatar_url приходят из auth.GetUsersProfiles
// (могут быть пустыми, если auth недоступен).
type TripParticipant struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username,omitempty"`
	AvatarURL    string `json:"avatar_url,omitempty"`
	PrivacyLevel string `json:"privacy_level"`
	Role         string `json:"role" example:"admin"`
}

// TripSettings — per-user настройки текущего юзера на трипе. Расширяемый bag:
// сюда добавляем новые per-user настройки (тема и т.п.) без новых эндпоинтов.
type TripSettings struct {
	NotificationsEnabled bool   `json:"notifications_enabled"`
	PrivacyLevel         string `json:"privacy_level"`
}

// ActiveAddMediaSession — активная сессия добавления медиа.
// CurrentInitiator — null до вызова AddMediaApplyGroupsAndProcess; заполняется
// обогащённым профилем (username + avatar_url) через auth.GetUsersProfiles.
type ActiveAddMediaSession struct {
	SessionID               string             `json:"session_id"`
	CurrentInitiator        *PublicUserProfile `json:"current_initiator,omitempty"`
	InitiatorAssignedAt     string             `json:"initiator_assigned_at,omitempty" example:"2026-04-25T12:00:00Z"`
	TakeoverAvailableAt     string             `json:"takeover_available_at,omitempty" example:"2026-04-25T13:00:00Z"`
	MediaCountInSession     int32              `json:"media_count_in_session"`
}

// PublicUserProfile — обогащённый публичный профиль участника (N2).
// avatar_url пустой, если аватар не загружен.
type PublicUserProfile struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// TripPin is a pin with media for trip view.
type TripPin struct {
	ID string `json:"id"`
	TripID string `json:"trip_id"`
	Name string `json:"name"`
	Description string `json:"description"`
	Category string `json:"category"`
	Latitude *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	StartTimeUnix int64 `json:"start_time_unix,omitempty"`
	EndTimeUnix int64 `json:"end_time_unix,omitempty"`
	PrivacyLevel string `json:"privacy_level"`
	Tags []string `json:"tags,omitempty"`
	Media []TripPinMedia `json:"media,omitempty"`
}

// TripPinMedia is media inside a pin.
type TripPinMedia struct {
	MediaID string `json:"media_id"`
	URL string `json:"url"`
	MediaType string `json:"media_type"`
	CapturedAtUnix int64 `json:"captured_at_unix,omitempty"`
	PrivacyLevel string `json:"privacy_level"`
}

// CreateTripResponse is the response for POST /api/v1/trips
type CreateTripResponse struct {
	TripID string `json:"trip_id"`
	Status string `json:"status"`
	UploadURLs []UploadURL `json:"upload_urls"`
}

// UploadURL is a single presigned URL (phase 3).
type UploadURL struct {
	ClientID string `json:"client_id"`
	S3Key string `json:"s3_key"`
	URL string `json:"url"`
}

// TripCoverUploadResponse — ответ POST /api/v1/trips/:id/cover/upload: presigned PUT URL + ключ для последующего confirm.
type TripCoverUploadResponse struct {
	UploadURL string `json:"upload_url"`
	S3Key string `json:"s3_key"`
}

// PrivacyResponse — ответ PUT /trips/{id}/privacy и аналогичных. privacy_level — эффективный (агрегированный) уровень после пересчёта.
type PrivacyResponse struct {
	PrivacyLevel string `json:"privacy_level"`
}

// GenerateInviteLinkResponse is the response for POST /api/v1/trips/:id/invite
type GenerateInviteLinkResponse struct {
	InviteLinkID string `json:"invite_link_id"`
	Token string `json:"token"`
	InviteURL string `json:"invite_url,omitempty"`
	ExpiresAtUnix int64 `json:"expires_at_unix"`
}

// JoinTripByTokenResponse is the response for POST /api/v1/trips/join
type JoinTripByTokenResponse struct {
	TripID string `json:"trip_id"`
	AlreadyJoined bool `json:"already_joined"`
}

// LeaveTripResponse is the response for POST /api/v1/trips/:id/leave
type LeaveTripResponse struct {
	Success bool `json:"success"`
	TripDeleted bool `json:"trip_deleted"`
}

// ProcessMediaGroupingResponse is the response for POST /api/v1/trips/creation/:id/media/process-grouping
type ProcessMediaGroupingResponse struct {
	TripID string `json:"trip_id"`
	Status string `json:"status"`
	DraftPins []DraftPin `json:"draft_pins"`
}

// DraftPin is one draft pin with media list.
type DraftPin struct {
	DraftPinID string `json:"draft_pin_id"`
	Media []DraftPinMedia `json:"media"`
}

// DraftPinMedia is media in a draft pin.
type DraftPinMedia struct {
	MediaID string `json:"media_id"`
	URL string `json:"url"`
	Type string `json:"type"`
}

// ApplyGroupsAndProcessResponse is the response for POST /api/v1/trips/creation/:id/apply-groups-and-process (202)
type ApplyGroupsAndProcessResponse struct {
	Message string `json:"message"`
	Status string `json:"status"`
}

// GetTripReviewResponse is the response for GET /api/v1/trips/creation/:id/review
type GetTripReviewResponse struct {
	TripID string `json:"trip_id"`
	Status string `json:"status"`
	Similar [][]string `json:"similar"` // groups of similar media IDs
	Pins []ReviewPin `json:"pins"`
}

// ReviewPin is a pin in the review step.
type ReviewPin struct {
	PinID string `json:"pin_id"`
	Name string `json:"name"`
	Category string `json:"category"`
	Latitude *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	LocationName string `json:"location_name,omitempty"`
	StartTimeUnix int64 `json:"start_time_unix,omitempty"`
	EndTimeUnix int64 `json:"end_time_unix,omitempty"`
	Issues []string `json:"issues,omitempty"`
	Tags []string `json:"tags,omitempty"`
	Media []ReviewPinMedia `json:"media,omitempty"`
}

// ReviewPinMedia is media in a review pin.
type ReviewPinMedia struct {
	MediaID string `json:"media_id"`
	URL string `json:"url"`
	PrivacyLevel string `json:"privacy_level"`
}

// FinalizeTripResponse is the response for POST /api/v1/trips/creation/:id/finalize
type FinalizeTripResponse struct {
	TripID string `json:"trip_id"`
	Status string `json:"status"`
	Message string `json:"message"`
}

// TripSettingsResponse is the response for PATCH /api/v1/trips/:id/settings .
type TripSettingsResponse struct {
	Success bool `json:"success"`
}

// FeedMedia is a lightweight media item for the feed card carousel .
type FeedMedia struct {
	MediaID string `json:"media_id"`
	URL string `json:"url"`
	MediaType string `json:"media_type"`
}

// FeedPin is a lightweight pin for the feed card map view .
type FeedPin struct {
	ID string `json:"id"`
	Latitude float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Media []FeedMedia `json:"media"`
}

// FeedItem is a single card in the feed: trip data + pins + media .
// is_liked / is_disliked / is_saved — per-user state текущего пользователя (ТЗ 7.5, 7.6).
type FeedItem struct {
	Trip Trip `json:"trip"`
	Pins []FeedPin `json:"pins"`
	Media []FeedMedia `json:"media"`
	IsLiked bool `json:"is_liked"`
	IsDisliked bool `json:"is_disliked"`
	IsSaved bool `json:"is_saved"`
}

// SuccessResponse is a generic success response for like/dislike/favourite.
type SuccessResponse struct {
	Success bool `json:"success"`
}

// AddMediaStartResponse is the response for POST /api/v1/trips/:id/media/add/start .
// Joined=true — подключились к уже активной сессии (race), upload_urls в этом
// случае пуст. Клиент запрашивает URLs через /request-upload-urls.
type AddMediaStartResponse struct {
	SessionID string `json:"session_id"`
	Status string `json:"status"`
	UploadURLs []UploadURL `json:"upload_urls"`
	Joined bool `json:"joined"`
}

// AddMediaRequestUploadUrlsResponse — presigned URLs для догрузки.
type AddMediaRequestUploadUrlsResponse struct {
	UploadURLs []UploadURL `json:"upload_urls"`
}

// AddMediaCommitUploadResponse — регистрация факта загрузки файла в S3.
type AddMediaCommitUploadResponse struct {
	MediaID string `json:"media_id"`
	MediaCountInSession int32 `json:"media_count_in_session"`
	RemainingSlots int32 `json:"remaining_slots"`
}

// SessionMediaEntry — одно медиа из снапшота add-media сессии.
type SessionMediaEntry struct {
	MediaID string `json:"media_id"`
	URL string `json:"url"`
	Type string `json:"type"`
	ActorUserID string `json:"actor_user_id,omitempty"`
	UploadedAt string `json:"uploaded_at" example:"2026-04-25T12:00:00Z"`
}

// AddMediaGetSessionMediaResponse — снимок медиа для экрана UPLOADING.
type AddMediaGetSessionMediaResponse struct {
	SessionID string `json:"session_id"`
	Media []SessionMediaEntry `json:"media"`
	MediaCountInSession int32 `json:"media_count_in_session"`
}

// AddMediaGetGroupingResponse — снимок draft_pins для экрана GROUPING_REVIEW.
type AddMediaGetGroupingResponse struct {
	TripID string `json:"trip_id"`
	SessionID string `json:"session_id"`
	DraftPins []DraftPin `json:"draft_pins"`
	ExistingMediaIDs []string `json:"existing_media_ids"`
}

// AddMediaGetReviewResponse — финальное ревью add-media.
// CanEdit=true для ведущего, а также для любого participant'а после истечения часа
// (первый мутирующий запрос автоматически перехватит инициативу).
// CurrentInitiator обогащён публичным профилем через auth.GetUsersProfiles (N2).
type AddMediaGetReviewResponse struct {
	TripID                  string             `json:"trip_id"`
	SessionID               string             `json:"session_id"`
	Pins                    []TripPin          `json:"pins"`
	NewPinIDs               []string           `json:"new_pin_ids"`
	ProtectedMediaIDs       []string           `json:"protected_media_ids"`
	CurrentInitiator        *PublicUserProfile `json:"current_initiator,omitempty"`
	TakeoverAvailableAt     string             `json:"takeover_available_at,omitempty" example:"2026-04-25T13:00:00Z"`
	CanEdit                 bool               `json:"can_edit"`
}

// AddMediaConfirmResponse — финализация сессии, трип → READY.
type AddMediaConfirmResponse struct {
	Status string `json:"status"`
	AlreadyConfirmed bool `json:"already_confirmed,omitempty"`
}

// AddMediaCancelResponse — отмена сессии, трип → READY.
type AddMediaCancelResponse struct {
	Status string `json:"status"`
}

// AddMediaTakeoverResponse — explicit перехват ведущего. is_initiator=true в любом
// успешном ответе (вы уже или только что стали ведущим). current_initiator
// обогащён профилем для отображения имени/аватара.
type AddMediaTakeoverResponse struct {
	IsInitiator         bool               `json:"is_initiator"`
	CurrentInitiator    *PublicUserProfile `json:"current_initiator,omitempty"`
	TakeoverAvailableAt string             `json:"takeover_available_at" example:"2026-04-25T13:00:00Z"`
}

// AddMediaProcessGroupingResponse is the response for POST /api/v1/trips/:id/media/add/process-grouping .
// ExistingMediaIDs marks the original media as read-only per spec 5.3.3.
type AddMediaProcessGroupingResponse struct {
	TripID string `json:"trip_id"`
	SessionID string `json:"session_id"`
	Status string `json:"status"`
	DraftPins []DraftPin `json:"draft_pins"`
	ExistingMediaIDs []string `json:"existing_media_ids"`
}

// AddMediaApplyGroupsAndProcessResponse is the response for POST /api/v1/trips/:id/media/add/apply-groups-and-process .
type AddMediaApplyGroupsAndProcessResponse struct {
	Message string `json:"message"`
	Status string `json:"status"`
}

// BattleMedia — одна карточка в выдаче StartBattle (ТЗ 8.1.1).
type BattleMedia struct {
	MediaID string `json:"media_id"`
	URL string `json:"url"`
	MediaType string `json:"media_type"`
}

// StartBattleResponse — ответ POST /api/v1/trips/:id/battles (ТЗ 8.1).
type StartBattleResponse struct {
	BattleID string `json:"battle_id"`
	Media []BattleMedia `json:"media"`
}

// SubmitBattleResultResponse — ответ POST /api/v1/trips/:id/battles/:battle_id/result (ТЗ 8.1.8).
type SubmitBattleResultResponse struct {
	NewBattleRating int32 `json:"new_battle_rating"`
}

// BestMemory — карточка медиа для story-mode "лучшие воспоминания" (ТЗ 8.2).
type BestMemory struct {
	MediaID string `json:"media_id"`
	URL string `json:"url"`
	MediaType string `json:"media_type"`
	BattleRating int32 `json:"battle_rating"`
	CapturedAtUnix int64 `json:"captured_at_unix"`
	PinName string `json:"pin_name"`
}

// GetBestMemoriesResponse — ответ GET /api/v1/trips/:id/best-memories (ТЗ 8.2).
type GetBestMemoriesResponse struct {
	Media []BestMemory `json:"media"`
}

// PinResponse — ответ GET/PATCH /api/v1/trips/{trip_id}/pins/{pin_id}
// и финального шага add-media сессии.
type PinResponse struct {
	Pin TripPin `json:"pin"`
}

// DeletePinResponse — ответ DELETE /api/v1/trips/{trip_id}/pins/{pin_id}.
// deletion_mode: "full" (ТЗ 4.5.1) — пин и медиа удалены физически;
// "soft_for_user" (ТЗ 4.5.2) — пин скрыт только для caller'а через pin_hidden_by_user.
type DeletePinResponse struct {
	DeletionMode string `json:"deletion_mode" example:"full"`
}

// PinUploadStartResponse — старт унифицированной pin-upload сессии.
type PinUploadStartResponse struct {
	SessionID  string      `json:"session_id"`
	UploadURLs []UploadURL `json:"upload_urls"`
}

// RequestPinUploadUrlsResponse — догрузка URL.
type RequestPinUploadUrlsResponse struct {
	UploadURLs []UploadURL `json:"upload_urls"`
}

// CommitPinUploadResponse — фиксация s3-загрузки.
type CommitPinUploadResponse struct {
	MediaID             string `json:"media_id"`
	MediaCountInSession int32  `json:"media_count_in_session"`
}

// ProcessPinUploadResponse — ответ async Process (HTTP 202).
type ProcessPinUploadResponse struct {
	SessionID        string `json:"session_id"`
	ProcessingStatus string `json:"processing_status" example:"PROCESSING"`
}

// PinSuggestedFields — suggested-поля нового пина (только для creation).
type PinSuggestedFields struct {
	Name          string   `json:"name"`
	Category      string   `json:"category"`
	Tags          []string `json:"tags,omitempty"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	StartTimeUnix *int64   `json:"start_time_unix,omitempty"`
	EndTimeUnix   *int64   `json:"end_time_unix,omitempty"`
}

// PinUploadDraft — snapshot Process. Suggested заполнено только для creation.
type PinUploadDraft struct {
	Suggested       *PinSuggestedFields `json:"suggested,omitempty"`
	Media           []ReviewPinMedia    `json:"media,omitempty"`
	PinIssues       []string            `json:"pin_issues,omitempty"`
	NSFWMediaIDs    []string            `json:"nsfw_media_ids,omitempty"`
	DedupedMediaIDs []string            `json:"deduped_media_ids,omitempty"`
}

// GetPinUploadReviewResponse — snapshot сессии. На UPLOADING/PROCESSING
// draft/similar пустые.
type GetPinUploadReviewResponse struct {
	SessionID        string         `json:"session_id"`
	ProcessingStatus string         `json:"processing_status" example:"READY_FOR_REVIEW"`
	Draft            PinUploadDraft `json:"draft"`
	Similar          [][]string     `json:"similar"`
}

// CancelPinUploadResponse — отмена pin-upload сессии.
type CancelPinUploadResponse struct {
	Status string `json:"status" example:"cancelled"`
}

// RecommendedPin — пин в карте популярных мест (ТЗ 9.3): один на кластер
// из топ-трипов региона, с превью-медиа из исходного пина.
type RecommendedPin struct {
	ID string `json:"id"`
	TripID string `json:"trip_id"`
	Latitude float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name string `json:"name"`
	Description string `json:"description"`
	Category string `json:"category"`
	LocationName string `json:"location_name"`
	MediaCount int32 `json:"media_count"`
	Media []FeedMedia `json:"media"`
}

// RecommendedMap — карта популярных мест региона. Поля trip и media заполнены так,
// чтобы карта могла отрисовываться в общей ленте как обычный FeedItem.
type RecommendedMap struct {
	RegionName string `json:"region_name"`
	RegionType string `json:"region_type"`
	Pins []RecommendedPin `json:"pins"`
	// Виртуальный трип-обёртка с name/description/dates/cover/счётчиками.
	// trip.id пуст до вызова SaveRecommendation.
	Trip Trip `json:"trip"`
	// Топ-8 медиа всей карты (карусель в карточке ленты).
	Media []FeedMedia `json:"media"`
	SnapshotToken string `json:"snapshot_token"`
}

// GetRecommendationsResponse — ответ GET /api/v1/recommendations.
type GetRecommendationsResponse struct {
	Map RecommendedMap `json:"map"`
}

type SaveRecommendationRequest struct {
	SnapshotToken string   `json:"snapshot_token"`
	PinIDs        []string `json:"pin_ids"`
	City          string   `json:"city"`
	Country       string   `json:"country"`
	Category      string   `json:"category"`
	Season        string   `json:"season"`
}

// SaveRecommendationResponse — ответ POST /api/v1/recommendations/save.
// Возвращает созданный generated-трип, добавленный в favourites.
type SaveRecommendationResponse struct {
	Trip Trip `json:"trip"`
}
