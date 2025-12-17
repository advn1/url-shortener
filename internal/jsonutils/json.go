package jsonutils

import (
	"bytes"
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		WriteInternalError(w)
		return
	}

	w.WriteHeader(statusCode)
	w.Write(buf.Bytes())
}
