package responses

type ErrorResponse struct {
	Error string `json:"error"`
	// Reason — машинный код ошибки (например, "SESSION_STALE", "NOT_INITIATOR",
	// "WRONG_STATUS", "LIMIT_EXCEEDED"). Клиент может различать сценарии и
	// реагировать по-разному, не парся текст Error.
	Reason string `json:"reason,omitempty"`
	// Details — дополнительные structured-поля для конкретного reason. Пример:
	// {"remaining_slots": 2, "limit": 500, "kind": "media"} для LIMIT_EXCEEDED.
	Details map[string]string `json:"details,omitempty"`
}
