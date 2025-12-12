package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/advn1/url-shortener/internal/jsonutils"
)

type GzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w GzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GzipMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// read gzip requests
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				jsonutils.WriteJSONError(w,http.StatusBadRequest, "bad gzip request", "")
				return
			}
			defer gz.Close()

			r.Body = gz
		}

		supportsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")

		if supportsGzip {
			w.Header().Set("Content-Encoding", "gzip")
			
			wr := gzip.NewWriter(w)
			gz := GzipWriter {ResponseWriter: w, Writer: wr}
			defer wr.Close()
			
			h.ServeHTTP(gz, r)
			return
		}

		h.ServeHTTP(w, r)
	})
}