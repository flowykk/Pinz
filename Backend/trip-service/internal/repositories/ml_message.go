package repositories

const (
	MLStreamTasks    = "ML_TASKS"
	MLStreamResults  = "ML_RESULTS"
	MLStreamTasksDLQ = "ML_TASKS_DLQ"

	MLSubjectTaskCreation         = "ml.tasks.creation"
	MLSubjectTaskAddMedia         = "ml.tasks.add_media"
	MLSubjectTaskPinUploadCreate  = "ml.tasks.pin_upload.creation"
	MLSubjectTaskPinUploadAddTo   = "ml.tasks.pin_upload.addition"
	MLSubjectTaskTextModeration   = "ml.tasks.text_moderation"

	MLSubjectResultPrefix = "ml.results."
	MLSubjectResultAll    = "ml.results.>"

	MLFlowCreation             = "creation"
	MLFlowAddMedia             = "add_media"
	MLFlowPinUploadCreation    = "pin_upload.creation"
	MLFlowPinUploadAddition    = "pin_upload.addition"
	MLFlowTextModeration       = "text_moderation"

	MLConsumerWorkers       = "ml-workers"
	MLConsumerTripResults   = "trip-results"

	MLTextEntityTrip  = "trip"
	MLTextEntityPin   = "pin"
	MLTextFieldName        = "name"
	MLTextFieldDescription = "description"
)

// IsNew=true → ML запускает NSFW; для existing медиа модерация повторно не нужна.
type MLMediaPayload struct {
	MediaID        string   `json:"media_id"`
	IsNew          bool     `json:"is_new"`
	MediaType      string   `json:"media_type"`
	S3Key          string   `json:"s3_key"`
	GetURL         string   `json:"get_url"`
	ContentType    string   `json:"content_type,omitempty"`
	CapturedAtUnix *int64   `json:"captured_at_unix,omitempty"`
	Latitude       *float64 `json:"latitude,omitempty"`
	Longitude      *float64 `json:"longitude,omitempty"`
}

type MLPinPayload struct {
	PinID       string           `json:"pin_id"`
	IsNew       bool             `json:"is_new"`
	Description string           `json:"description,omitempty"`
	Media       []MLMediaPayload `json:"media"`
}

type MLTaskMessage struct {
	Flow          string         `json:"flow"`
	TripID        string         `json:"trip_id"`
	SessionID     string         `json:"session_id,omitempty"`
	TargetPinID   string         `json:"target_pin_id,omitempty"`
	Pins          []MLPinPayload `json:"pins"`
	ExpiresAtUnix int64          `json:"expires_at_unix"`
}

type MLPinSuggestion struct {
	PinID    string   `json:"pin_id"`
	Category string   `json:"category"`
	Tags     []string `json:"tags,omitempty"`
}

type MLResultMessage struct {
	Flow           string             `json:"flow"`
	TripID         string             `json:"trip_id"`
	SessionID      string             `json:"session_id,omitempty"`
	SimilarGroups  [][]string         `json:"similar_groups,omitempty"`
	NSFWIDs        []string           `json:"nsfw_ids,omitempty"`
	PinSuggestions []MLPinSuggestion  `json:"pin_suggestions,omitempty"`
	TextResults    []MLTextItemResult `json:"text_results,omitempty"`
}

type MLTextItem struct {
	ItemID     string `json:"item_id"`
	EntityKind string `json:"entity_kind"`
	EntityID   string `json:"entity_id"`
	Field      string `json:"field"`
	Text       string `json:"text"`
}

type MLTextTaskMessage struct {
	Flow          string       `json:"flow"`
	TripID        string       `json:"trip_id,omitempty"`
	Items         []MLTextItem `json:"items"`
	ExpiresAtUnix int64        `json:"expires_at_unix"`
}

type MLTextItemResult struct {
	ItemID     string `json:"item_id"`
	EntityKind string `json:"entity_kind"`
	EntityID   string `json:"entity_id"`
	Field      string `json:"field"`
	Censored   bool   `json:"censored"`
}

func SubjectForFlow(flow string) string {
	switch flow {
	case MLFlowCreation:
		return MLSubjectTaskCreation
	case MLFlowAddMedia:
		return MLSubjectTaskAddMedia
	case MLFlowPinUploadCreation:
		return MLSubjectTaskPinUploadCreate
	case MLFlowPinUploadAddition:
		return MLSubjectTaskPinUploadAddTo
	case MLFlowTextModeration:
		return MLSubjectTaskTextModeration
	default:
		return ""
	}
}
