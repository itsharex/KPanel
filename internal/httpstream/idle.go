package httpstream

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

type idleReader struct {
	ctx        context.Context
	controller *http.ResponseController
	reader     io.Reader
	timeout    time.Duration
}

func NewIdleReader(
	ctx context.Context,
	writer http.ResponseWriter,
	reader io.Reader,
	timeout time.Duration,
) io.Reader {
	return &idleReader{
		ctx:        ctx,
		controller: http.NewResponseController(writer),
		reader:     reader,
		timeout:    timeout,
	}
}

func (reader *idleReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	if err := reader.controller.SetReadDeadline(time.Now().Add(reader.timeout)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		return 0, err
	}
	return reader.reader.Read(buffer)
}

type IdleResponseWriter struct {
	http.ResponseWriter
	ctx        context.Context
	controller *http.ResponseController
	timeout    time.Duration
}

func NewIdleResponseWriter(
	ctx context.Context,
	writer http.ResponseWriter,
	timeout time.Duration,
) *IdleResponseWriter {
	return &IdleResponseWriter{
		ResponseWriter: writer,
		ctx:            ctx,
		controller:     http.NewResponseController(writer),
		timeout:        timeout,
	}
}

func (writer *IdleResponseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}

func (writer *IdleResponseWriter) WriteHeader(statusCode int) {
	_ = writer.controller.SetWriteDeadline(time.Now().Add(writer.timeout))
	writer.ResponseWriter.WriteHeader(statusCode)
}

func (writer *IdleResponseWriter) Write(buffer []byte) (int, error) {
	if err := writer.ctx.Err(); err != nil {
		return 0, err
	}
	if err := writer.controller.SetWriteDeadline(time.Now().Add(writer.timeout)); err != nil &&
		!errors.Is(err, http.ErrNotSupported) {
		return 0, err
	}
	return writer.ResponseWriter.Write(buffer)
}
