package io_utils

import "io"

type CountingWriter struct {
	Writer       io.Writer
	BytesWritten int64
}

func NewCountingWriter(writer io.Writer) *CountingWriter {
	return &CountingWriter{Writer: writer}
}

func (cw *CountingWriter) Write(p []byte) (n int, err error) {
	n, err = cw.Writer.Write(p)
	cw.BytesWritten += int64(n)
	return n, err
}

func (cw *CountingWriter) GetBytesWritten() int64 {
	return cw.BytesWritten
}
