package retry

import (
	"bytes"
	"net/http"
)

type ResponseBuffer struct {
	header      http.Header
	statusCode  int
	body        bytes.Buffer
	wroteHeader bool
}

func NewResponseBuffer() *ResponseBuffer {
	return &ResponseBuffer{
		header:     make(http.Header),
		statusCode: http.StatusOK,
	}
}

func (b *ResponseBuffer) Header() http.Header {
	return b.header
}

func (b *ResponseBuffer) WriteHeader(statusCode int) {
	if b.wroteHeader {
		return
	}
	b.statusCode = statusCode
	b.wroteHeader = true
}

func (b *ResponseBuffer) Write(p []byte) (int, error) {
	if !b.wroteHeader {
		b.WriteHeader(http.StatusOK)
	}
	return b.body.Write(p)
}

func (b *ResponseBuffer) StatusCode() int {
	return b.statusCode
}

func (b *ResponseBuffer) WriteTo(w http.ResponseWriter) {
	for key, vals := range b.header {
		for _, val := range vals {
			w.Header().Add(key, val)
		}
	}
	w.WriteHeader(b.statusCode)
	_, _ = w.Write(b.body.Bytes())
}
