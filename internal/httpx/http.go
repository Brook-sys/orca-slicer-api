package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func NewError(status int, message string) *Error {
	return &Error{Status: status, Message: message}
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func WriteError(w http.ResponseWriter, err error) {
	var httpError *Error
	if errors.As(err, &httpError) {
		WriteJSON(w, httpError.Status, map[string]string{"message": httpError.Message})
		return
	}
	slog.Error("internal server error", "error", err)
	WriteJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
}
