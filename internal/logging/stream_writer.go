package logging

import (
	"bytes"
	"io"
	"unicode/utf8"
)

// maxStreamBuffer caps how much unterminated text a StreamWriter holds
// back while waiting for a line boundary. A longer line is flushed at
// the cap so memory stays bounded and long-line streams still move.
const maxStreamBuffer = 64 << 10

// StreamWriter redacts a byte stream that arrives in arbitrary Write
// chunks (child process stdio). Unlike NewWriter, which redacts each
// Write in isolation, it holds back the trailing partial line so a
// secret split across two Writes is still seen whole.
//
// Rules:
//   - Redaction disabled: bytes pass straight through, nothing held.
//   - Text is released up to the last '\n' or '\r' boundary; the
//     remainder waits for more data, the cap, or Close.
//   - Segments containing NUL or invalid UTF-8 are binary and pass
//     through untouched, so non-text streams stay byte-identical.
//
// Close MUST be called when the stream ends to flush the held tail.
// Not safe for concurrent Write.
type StreamWriter struct {
	inner io.Writer
	buf   []byte
}

// NewStreamWriter returns a boundary-safe redacting writer over w.
func NewStreamWriter(w io.Writer) *StreamWriter {
	if w == nil {
		w = io.Discard
	}
	return &StreamWriter{inner: w}
}

// Write implements io.Writer. It always reports len(p) consumed on
// success, regardless of redaction length changes or held bytes.
func (s *StreamWriter) Write(p []byte) (int, error) {
	if !Enabled() {
		if err := s.flush(false); err != nil {
			return 0, err
		}
		if _, err := s.inner.Write(p); err != nil {
			return 0, err
		}
		return len(p), nil
	}
	s.buf = append(s.buf, p...)
	if i := bytes.LastIndexAny(s.buf, "\r\n"); i >= 0 {
		if err := s.emit(s.buf[:i+1]); err != nil {
			return 0, err
		}
		s.buf = append(s.buf[:0], s.buf[i+1:]...)
	}
	for len(s.buf) > maxStreamBuffer {
		n := maxStreamBuffer
		// Do not cut a multi-byte rune in half.
		for n > 0 && !utf8.RuneStart(s.buf[n]) {
			n--
		}
		if n == 0 {
			n = maxStreamBuffer
		}
		if err := s.emit(s.buf[:n]); err != nil {
			return 0, err
		}
		s.buf = append(s.buf[:0], s.buf[n:]...)
	}
	return len(p), nil
}

// Close flushes the held tail. It does not close the inner writer.
func (s *StreamWriter) Close() error {
	return s.flush(true)
}

func (s *StreamWriter) flush(redact bool) error {
	if len(s.buf) == 0 {
		return nil
	}
	seg := s.buf
	s.buf = nil
	if redact && Enabled() {
		return s.emit(seg)
	}
	_, err := s.inner.Write(seg)
	return err
}

// emit writes seg, redacted unless it is binary.
func (s *StreamWriter) emit(seg []byte) error {
	if len(seg) == 0 {
		return nil
	}
	if bytes.IndexByte(seg, 0) < 0 && utf8.Valid(seg) {
		seg = Redactor().ApplyBytes(seg)
	}
	_, err := s.inner.Write(seg)
	return err
}
