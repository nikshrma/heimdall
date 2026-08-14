package retry

import "net/http"

type Policy struct {
	MaxAttempts int32
}

func ShouldRetryMethod(method string) bool {
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return true
	}
	return false
}

func ShouldRetryStatus(status int) bool {
	if status >= 502 && status <= 504 {
		return true
	}
	return false
}
