package jsonutils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type JSONError struct {
	StatusCode int    `json:"status_code"`
	Err        string `json:"error"`
	Msg        string `json:"message"`
}

func (j JSONError) Error() string {
	return fmt.Sprintf("%d: %s - %s", j.StatusCode, j.Err, j.Msg)
}

func NewJSONError(statusCode int, err string, msg string) JSONError {
	return JSONError{StatusCode: statusCode, Err: err, Msg: msg}
}

func WriteJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("json encode error: %v\n", err)
	}
}

func WriteJSONError(w http.ResponseWriter, j JSONError) {
	WriteJSON(w, j.StatusCode, j)
}

func BadRequest(w http.ResponseWriter, msg string) {
	err := NewJSONError(http.StatusBadRequest, "Bad Request", msg)
	WriteJSON(w, http.StatusBadRequest, err)
}

func InternalError(w http.ResponseWriter) {
	err := NewJSONError(http.StatusInternalServerError, "Internal Server Error", "")
	WriteJSON(w, http.StatusInternalServerError, err)
}
