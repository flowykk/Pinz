package services

import "regexp"

var usernameRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func validateUsernameFormat(username string) bool {
	return usernameRegex.MatchString(username)
}
