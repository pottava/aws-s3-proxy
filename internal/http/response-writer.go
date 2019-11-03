package http

import (
	"io"
	"net/http"
)

type custom struct {
	io.Writer
	http.ResponseWriter
	status int
}

func (c *custom) Write(b []byte) (int, error) {
	if c.Header().Get("Content-Type") == "" {
		c.Header().Set("Content-Type", http.DetectContentType(b))
	}
	return c.Writer.Write(b)
}

func (c *custom) WriteHeader(status int) {
	c.ResponseWriter.WriteHeader(status)
	c.status = status
}
