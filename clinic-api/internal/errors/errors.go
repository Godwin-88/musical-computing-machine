package errors

import "net/http"

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

func NotFound(msg string) (int, ErrorResponse) {
	return http.StatusNotFound, ErrorResponse{
		Error: APIError{Code: "NOT_FOUND", Message: msg},
	}
}

func Conflict(msg string) (int, ErrorResponse) {
	return http.StatusConflict, ErrorResponse{
		Error: APIError{Code: "CONFLICT", Message: msg},
	}
}

func Validation(msg string) (int, ErrorResponse) {
	return http.StatusUnprocessableEntity, ErrorResponse{
		Error: APIError{Code: "VALIDATION_ERROR", Message: msg},
	}
}

func Unauthorized(msg string) (int, ErrorResponse) {
	return http.StatusUnauthorized, ErrorResponse{
		Error: APIError{Code: "UNAUTHORIZED", Message: msg},
	}
}

func Internal() (int, ErrorResponse) {
	return http.StatusInternalServerError, ErrorResponse{
		Error: APIError{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred."},
	}
}

func ServiceUnavailable(msg string) (int, ErrorResponse) {
	return http.StatusServiceUnavailable, ErrorResponse{
		Error: APIError{Code: "SERVICE_UNAVAILABLE", Message: msg},
	}
}