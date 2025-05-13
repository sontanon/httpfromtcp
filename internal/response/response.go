package response

import (
	"fmt"
	"httpfromtcp/internal/headers"
	"io"
	"log"
)

type StatusCode int

const (
	StatusCodeOK                  StatusCode = 200
	StatusCodeBadRequest          StatusCode = 400
	StatusCodeInternalServerError StatusCode = 500
)

type writerState int

const (
	writerStateStatusLine = iota
	writerStateHeaders
	writerStateBody
	writerStateTrailers
	writerStateDone
)

const CRLF = "\r\n"

type Writer struct {
	io.Writer
	state writerState
}

func NewWriter(w io.Writer) Writer {
	return Writer{w, writerStateStatusLine}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.state != writerStateStatusLine {
		return fmt.Errorf("invalid state: %v", w.state)
	}

	var reasonPhrase string
	switch statusCode {
	case StatusCodeOK:
		reasonPhrase = "OK"
	case StatusCodeBadRequest:
		reasonPhrase = "Bad Request"
	case StatusCodeInternalServerError:
		reasonPhrase = "Internal Server Error"
	default:
		reasonPhrase = ""
	}

	statusLine := fmt.Sprintf("HTTP/1.1 %d %s%s", statusCode, reasonPhrase, CRLF)

	// log.Println(statusLine)
	_, err := w.Write([]byte(statusLine))
	w.state = writerStateHeaders
	return err
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	h := headers.NewHeaders()
	h.Set("Content-Length", fmt.Sprintf("%d", contentLen))
	h.Set("Connection", "close")
	h.Set("Content-Type", "text/plain")

	return h
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
	if w.state != writerStateHeaders {
		return fmt.Errorf("invalid state %v", w.state)
	}
	for key, value := range headers {

		line := fmt.Sprintf("%s: %s%s", key, value, CRLF)
		log.Println(line)

		if _, err := w.Write([]byte(line)); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte(CRLF)); err != nil {
		return err
	}

	w.state = writerStateBody
	return nil
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	if w.state != writerStateBody {
		return 0, fmt.Errorf("invalid state %v", w.state)
	}
	n, err := w.Write(p)
	w.state = writerStateDone
	return n, err
}

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	if w.state != writerStateBody {
		return 0, fmt.Errorf("invalid state %v", w.state)
	}

	n := len(p)
	return w.Write([]byte(fmt.Sprintf("%X%s%s%s", n, CRLF, p, CRLF)))
}

func (w *Writer) WriteChunkedBodyDone() (int, error) {
	if w.state != writerStateBody {
		return 0, fmt.Errorf("invalid state %v", w.state)
	}

	w.state = writerStateTrailers
	return w.Write([]byte(fmt.Sprintf("0%s", CRLF)))
}

func (w *Writer) WriteTrailers(headers headers.Headers) error {
	if w.state != writerStateTrailers {
		return fmt.Errorf("invalid state %v", w.state)
	}

	for key, value := range headers {

		line := fmt.Sprintf("%s: %s%s", key, value, CRLF)
		log.Println(line)

		if _, err := w.Write([]byte(line)); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte(CRLF)); err != nil {
		return err
	}

	w.state = writerStateDone
	return nil
}
