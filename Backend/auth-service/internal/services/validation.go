package services

import "regexp"

var usernameRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// validateUsernameFormat checks that the username contains only allowed characters.
func validateUsernameFormat(username string) bool {
	return usernameRegex.MatchString(username)
}
