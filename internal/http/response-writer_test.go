package http

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteText(t *testing.T) {
	expected := "text/plain; charset=utf-8"

	w := httptest.NewRecorder()
	c := custom{Writer: w, ResponseWriter: w}
	n, err := c.Write([]byte("hello"))

	assert.Nil(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, expected, c.Header().Get("Content-Type"))
}

func TestWriteHTML(t *testing.T) {
	expected := "text/html; charset=utf-8"

	w := httptest.NewRecorder()
	c := custom{Writer: w, ResponseWriter: w}
	n, err := c.Write([]byte("<html><body>hello</body></html>"))

	assert.Nil(t, err)
	assert.Equal(t, 31, n)
	assert.Equal(t, expected, c.Header().Get("Content-Type"))
}

func TestWritePDF(t *testing.T) {
	expected := "application/pdf"

	w := httptest.NewRecorder()
	c := custom{Writer: w, ResponseWriter: w}
	n, err := c.Write([]byte("%PDF-test"))

	assert.Nil(t, err)
	assert.Equal(t, 9, n)
	assert.Equal(t, expected, c.Header().Get("Content-Type"))
}

func TestWriteHeader(t *testing.T) {
	expected := 200

	w := httptest.NewRecorder()
	c := custom{ResponseWriter: w}
	c.WriteHeader(expected)

	assert.Equal(t, expected, c.status)
	assert.Equal(t, expected, w.Result().StatusCode)
}
