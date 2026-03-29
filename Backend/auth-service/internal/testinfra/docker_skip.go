package testinfra

import (
	"strings"
)

// isInfraStartUnavailable is true when container start failed due to Docker/connectivity.
func isInfraStartUnavailable(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	lower := strings.ToLower(s)
	return strings.Contains(s, "Cannot connect to the Docker daemon") ||
		strings.Contains(lower, "docker daemon") ||
		strings.Contains(s, "Is the docker daemon running") ||
		strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "docker.sock") && strings.Contains(lower, "no such file")
}
