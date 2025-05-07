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
	StatusCodeInternalServerError            = 500
)

const CRLF = "\r\n"

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
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

	log.Println(statusLine)
	_, err := w.Write([]byte(statusLine))
	return err
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	h := headers.NewHeaders()
	h["Content-Length"] = fmt.Sprintf("%d", contentLen)
	h["Connection"] = "close"
	h["Content-Type"] = "text/plain"

	return h
}

func WriteHeaders(w io.Writer, headers headers.Headers) error {
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
	return nil
}
