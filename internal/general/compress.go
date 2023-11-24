package general

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"strings"
)

type GzipWriter struct {
	OldW   http.ResponseWriter
	Writer *gzip.Writer
}

func (w GzipWriter) WriteHeader(statusCode int) {
	w.OldW.WriteHeader(statusCode)
}

func (w GzipWriter) Header() http.Header {
	return w.OldW.Header()
}

func (w GzipWriter) Write(b []byte) (int, error) {
	contentType := w.OldW.Header().Get("Content-Type")
	fmt.Printf("Content-Type of response: '%s'\n", contentType)
	if !strings.Contains(contentType, "application/json") &&
		!strings.Contains(contentType, "text/html") {
		fmt.Println("Response answer is", string(b))
		return w.OldW.Write(b)
	}
	fmt.Println("Started encoding...")
	w.OldW.Header().Set("Content-Encoding", "gzip")
	w.WriteHeader(http.StatusOK)
	return w.Writer.Write(b)
}
