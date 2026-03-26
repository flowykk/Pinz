package services

type MediaURLResolver interface {
	// PresignedUploadURL возвращает URL для загрузки файла в объектное хранилище.
	PresignedUploadURL(s3Key string) string
	// ReadURL возвращает URL для чтения медиа (используется в review/draft ответах).
	ReadURL(s3Key string) string
}
