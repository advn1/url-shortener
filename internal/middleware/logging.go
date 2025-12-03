package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type ResponseData struct {
	StatusCode int
	Size       int
}

// custom status recorder. saves response data (status code and size)
type StatusRecorder struct {
	http.ResponseWriter
	responseData ResponseData
}

// implements ResponseWriter (Write, WriteHeader)
func (r *StatusRecorder) Write(b []byte) (int, error) {
	if r.responseData.StatusCode == 0 {
		r.responseData.StatusCode = http.StatusOK
	}
	size, err := r.ResponseWriter.Write(b)
	r.responseData.Size += size
	return size, err
}

func (r *StatusRecorder) WriteHeader(statusCode int) {
	r.responseData.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func LoggingMiddleware(h http.HandlerFunc, sugar *zap.SugaredLogger) http.HandlerFunc {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestURI := r.RequestURI
		method := r.Method

		rd := ResponseData{Size: 0, StatusCode: 0}

		lw := &StatusRecorder{ResponseWriter: w, responseData: rd}

		h.ServeHTTP(lw, r)

		duration := time.Since(start)

		sugar.Infow("Request Info", "URI", requestURI, "method", method, "duration", duration)
		sugar.Infow("Response Info", "status", lw.responseData.StatusCode, "size", lw.responseData.Size)
	}

	return http.HandlerFunc(logFn)
}
