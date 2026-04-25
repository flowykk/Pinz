package requests

// CreateTripRequest is the REST body for POST /api/v1/trips (creation/start). Privacy задаётся per-user через PUT /trips/{id}/privacy.
type CreateTripRequest struct {
	Name string `json:"name"`
	Description string `json:"description"`
	Category string `json:"category"`
	Season string `json:"season"`
	FilesToUpload []FileToUploadEntry `json:"files_to_upload,omitempty"`
}

// FileToUploadEntry is one file to upload (client_id + content_type for Presigned URL).
type FileToUploadEntry struct {
	ClientID string `json:"client_id"`
	ContentType string `json:"content_type"`
}

// UpdateTripRequest is the REST body for PATCH /api/v1/trips/:id.
// Обложка редактируется отдельно: POST /cover/upload → PUT в S3 → POST /cover/confirm (DELETE /cover для очистки).
// Privacy задаётся per-user через PUT /api/v1/trips/{id}/privacy.
type UpdateTripRequest struct {
	Name *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Category *string `json:"category,omitempty"`
	Season *string `json:"season,omitempty"`
	StartDateUnix *int64 `json:"start_date_unix,omitempty"`
	EndDateUnix *int64 `json:"end_date_unix,omitempty"`
}

// RequestTripCoverUploadRequest — тело POST /api/v1/trips/:id/cover/upload (step 1 двухшагового потока обложки).
type RequestTripCoverUploadRequest struct {
	Filename string `json:"filename" example:"cover.jpg"`
	ContentType string `json:"content_type" example:"image/jpeg"`
}

// ConfirmTripCoverUploadRequest — тело POST /api/v1/trips/:id/cover/confirm (step 2, s3_key из /cover/upload).
type ConfirmTripCoverUploadRequest struct {
	S3Key string `json:"s3_key"`
}

// GenerateInviteLinkRequest is the REST body for POST /api/v1/trips/:id/invite
type GenerateInviteLinkRequest struct {
	ExpiresInSeconds *int64 `json:"expires_in_seconds,omitempty"`
}

// JoinTripByTokenRequest is the REST body for POST /api/v1/trips/join
type JoinTripByTokenRequest struct {
	Token string `json:"token"`
}

// ProcessMediaGroupingRequest is the REST body for POST /api/v1/trips/creation/:id/media/process-grouping
type ProcessMediaGroupingRequest struct {
	Media []MediaMetaEntry `json:"media"`
}

// MediaMetaEntry is metadata for one uploaded media file.
type MediaMetaEntry struct {
	S3Key string `json:"s3_key"`
	MediaType string `json:"media_type"`
	CapturedAt string `json:"captured_at,omitempty"` // ISO8601
	Latitude *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	ContentHash *string `json:"content_hash,omitempty"` // e.g. SHA-256 for duplicate detection
}

// ApplyGroupsAndProcessRequest is the REST body for POST /api/v1/trips/creation/:id/apply-groups-and-process
type ApplyGroupsAndProcessRequest struct {
	DraftPins []DraftPinInput `json:"draft_pins"`
	DeletedMediaIDs []string `json:"deleted_media_ids,omitempty"`
}

// DraftPinInput is one draft pin with media IDs.
type DraftPinInput struct {
	DraftPinID string `json:"draft_pin_id"`
	MediaIDs []string `json:"media_ids"`
}

// FinalizeTripRequest is the REST body for POST /api/v1/trips/creation/:id/finalize
type FinalizeTripRequest struct {
	PinUpdates []PinUpdateInput `json:"pin_updates,omitempty"`
	MediaToDelete []string `json:"media_to_delete,omitempty"`
}

// PublishTripRequest is the REST body for POST /api/v1/trips/:id/publish.
type PublishTripRequest struct {
	PublishWhole bool `json:"publish_whole,omitempty"`
	PinIDs []string `json:"pin_ids,omitempty"`
}

// PinUpdateInput is name and/or manual coordinates for a pin.
type PinUpdateInput struct {
	PinID string `json:"pin_id"`
	Name *string `json:"name,omitempty"`
	Latitude *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	Description *string `json:"description,omitempty"`
	Category *string `json:"category,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

// UpdatePinRequest — тело PATCH /api/v1/trips/{trip_id}/pins/{pin_id}.
// Privacy задаётся per-user через PUT /trips/{trip_id}/pins/{pin_id}/privacy.
// tags применяются как replace-all только при tags_set=true (даже пустым массивом).
type UpdatePinRequest struct {
	Name *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Category *string `json:"category,omitempty"`
	Latitude *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	StartTimeUnix *int64 `json:"start_time_unix,omitempty"`
	EndTimeUnix *int64 `json:"end_time_unix,omitempty"`
	Tags []string `json:"tags,omitempty"`
	TagsSet bool `json:"tags_set,omitempty"`
}

// AddMediaToPinStartRequest — тело POST /api/v1/trips/{trip_id}/pins/{pin_id}/media/sessions/start.
type AddMediaToPinStartRequest struct {
	FilesToUpload []FileToUploadEntry `json:"files_to_upload"`
}

// RequestPinMediaUploadUrlsRequest — догрузка presigned URLs к активной сессии.
type RequestPinMediaUploadUrlsRequest struct {
	FilesToUpload []FileToUploadEntry `json:"files_to_upload"`
}

// CommitPinMediaUploadRequest — клиент зовёт после успешного PUT в S3.
// Сервер создаёт media entry с pin_id=NULL и pin_addition_session_id=session.
type CommitPinMediaUploadRequest struct {
	S3Key string `json:"s3_key"`
	MediaType string `json:"media_type"` // image | video
	CapturedAtUnix *int64 `json:"captured_at_unix,omitempty"`
	Latitude *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

// FinalizePinMediaAdditionRequest — финализация добавления медиа в пин (ТЗ 4.13-4.14).
type FinalizePinMediaAdditionRequest struct {
	MediaToDelete []string `json:"media_to_delete,omitempty"`
}

// CreatePinStartRequest — старт sessioned-флоу создания пина (ТЗ 4.1, 4.6).
type CreatePinStartRequest struct {
	FilesToUpload []FileToUploadEntry `json:"files_to_upload"`
}

// RequestPinCreationUploadUrlsRequest — догрузка presigned URLs к активной сессии.
type RequestPinCreationUploadUrlsRequest struct {
	FilesToUpload []FileToUploadEntry `json:"files_to_upload"`
}

// CommitPinCreationUploadRequest — фиксация s3-загрузки в pin-creation сессию.
type CommitPinCreationUploadRequest struct {
	S3Key string `json:"s3_key"`
	MediaType string `json:"media_type"` // image | video
	CapturedAtUnix *int64 `json:"captured_at_unix,omitempty"`
	Latitude *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

// FinalizePinCreationRequest — финал создания пина (ТЗ 4.9-4.11).
// Если поле nil/пусто — берётся suggested значение из snapshot ProcessPinCreation.
// tags применяется как replace-all только при tags_set=true.
type FinalizePinCreationRequest struct {
	Name *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Category *string `json:"category,omitempty"`
	Latitude *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	StartTimeUnix *int64 `json:"start_time_unix,omitempty"`
	EndTimeUnix *int64 `json:"end_time_unix,omitempty"`
	Tags []string `json:"tags,omitempty"`
	TagsSet bool `json:"tags_set,omitempty"`
	MediaToDelete []string `json:"media_to_delete,omitempty"`
}

// UpsertPrivacyRequest — тело PUT /trips/{id}/privacy и аналогичных ручек по pin/media.
type UpsertPrivacyRequest struct {
	PrivacyLevel string `json:"privacy_level"`
}

// AddMediaStartRequest is the REST body for POST /api/v1/trips/:id/media/add/start
type AddMediaStartRequest struct {
	FilesToUpload []FileToUploadEntry `json:"files_to_upload,omitempty"`
}

// AddMediaProcessGroupingRequest is the REST body for POST /api/v1/trips/:id/media/add/process-grouping.
// media[] больше не передаётся (медиа регистрируются через /commit-upload).
// add_more=true откатывает GROUPING_REVIEW → UPLOADING для докидывания файлов.
type AddMediaProcessGroupingRequest struct {
	SessionID string `json:"session_id"`
	AddMore bool `json:"add_more,omitempty"`
}

// AddMediaApplyGroupsAndProcessRequest is the REST body for POST /api/v1/trips/:id/media/add/apply-groups-and-process
type AddMediaApplyGroupsAndProcessRequest struct {
	SessionID string `json:"session_id"`
	DraftPins []DraftPinInput `json:"draft_pins"`
	DeletedMediaIDs []string `json:"deleted_media_ids,omitempty"`
}

// AddMediaRequestUploadUrlsRequest — запрос presigned URL для догрузки файлов
// в уже активную сессию (клиент присоединился, либо хочет добавить ещё файлов).
type AddMediaRequestUploadUrlsRequest struct {
	SessionID string `json:"session_id"`
	FilesToUpload []FileToUploadEntry `json:"files_to_upload"`
}

// AddMediaCommitUploadRequest — клиент зовёт после каждого успешного PUT в S3.
// Сервер создаёт media entry (pin_id NULL) и публикует WS ADD_MEDIA_PROGRESS.
type AddMediaCommitUploadRequest struct {
	SessionID string `json:"session_id"`
	S3Key string `json:"s3_key"`
	MediaType string `json:"media_type"`
	CapturedAt string `json:"captured_at,omitempty"` // RFC3339
	Latitude *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

// AddMediaConfirmRequest — финализация add-media сессии: правки пинов и удаление
// «похожих» медиа передаются батчем и применяются атомарно с закрытием сессии
// (ТЗ 3.14/3.15). Аналогично FinalizeTripRequest для creation-флоу. Если pin_updates
// и media_to_delete пусты — Confirm просто подтверждает сессию без правок.
// session_id нужен для защиты от устаревшего клиентского контекста.
type AddMediaConfirmRequest struct {
	SessionID     string           `json:"session_id"`
	PinUpdates    []PinUpdateInput `json:"pin_updates,omitempty"`
	MediaToDelete []string         `json:"media_to_delete,omitempty"`
}
type AddMediaCancelRequest struct {
	SessionID string `json:"session_id"`
}

// AddMediaTakeoverRequest — тело POST /api/v1/trips/:id/media/add/takeover.
// Идемпотентен: если caller уже ведущий — 200 без изменений.
type AddMediaTakeoverRequest struct {
	SessionID string `json:"session_id"`
}

// TripSettingsRequest is the REST body for PATCH /api/v1/trips/:id/settings.
type TripSettingsRequest struct {
	NotificationsEnabled bool `json:"notifications_enabled"`
}

// SubmitBattleResultRequest — тело POST /api/v1/trips/:id/battles/:battle_id/result (ТЗ 8.1.8).
type SubmitBattleResultRequest struct {
	WinnerMediaID string `json:"winner_media_id"`
}
