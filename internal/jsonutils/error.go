package jsonutils

import (
	"errors"
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

func WriteJSONError(w http.ResponseWriter, j JSONError) {
	WriteJSON(w, j.StatusCode, j)
}

func WriteHTTPError(w http.ResponseWriter, err error) {
	var jsonError JSONError
	if errors.As(err, &jsonError) {
		WriteJSONError(w, jsonError)
	} else {
		WriteInternalError(w)
	}
}

func WriteBadRequest(w http.ResponseWriter, msg string) {
	err := BadRequest(msg)
	WriteJSONError(w, err.(JSONError))
}

func WriteInternalError(w http.ResponseWriter) {
	err := InternalError()
	WriteJSONError(w, err.(JSONError))
}

func BadRequest(msg string) error {
	err := NewJSONError(http.StatusBadRequest, "Bad Request", msg)
	return err
}

func InternalError() error {
	err := NewJSONError(http.StatusInternalServerError, "Internal Server Error", "")
	return err
}
