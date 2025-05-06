package request

import (
	"bytes"
	"errors"
	"fmt"
	"httpfromtcp/internal/headers"
	"io"
	"strconv"
	"strings"
	"unicode"
)

const BUFFER_SIZE int = 8
const CRLF = "\r\n"

type parserState int

const (
	parserStateInvalid parserState = iota
	parserStateInitialized
	parserStateParsingHeaders
	parserStateParsingBody
	parserStateDone
)

type Request struct {
	RequestLine   RequestLine
	Headers       headers.Headers
	state         parserState
	Body          []byte
	bodyBytesRead int
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func (r Request) PrettyPrint() string {
	headersString := ""
	for key, value := range r.Headers {
		headersString += fmt.Sprintf("- %s: %s\n", key, value)
	}
	bodyString := ""
	if r.Body != nil {
		bodyString = fmt.Sprintf("\nBody:\n%s\n", string(r.Body))
	}
	return fmt.Sprintf(`Request line:
- Method: %s
- Target: %s
- Version: 1.1
Headers:
%s%s`, r.RequestLine.Method, r.RequestLine.RequestTarget, headersString, bodyString)
}

func (r *Request) parse(data []byte) (int, error) {
	switch r.state {
	case parserStateInitialized:
		requestLine, n, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, nil
		}
		r.RequestLine = *requestLine
		r.state = parserStateParsingHeaders
		return n, nil
	case parserStateParsingHeaders:
		n, done, err := r.Headers.Parse(data)
		if err != nil {
			return 0, err
		}
		if done {
			r.state = parserStateParsingBody
		}
		return n, nil
	case parserStateParsingBody:
		if r.Body == nil {
			contentLengthString, exists := r.Headers.Get("Content-Length")
			if !exists {
				r.state = parserStateDone
				return 0, nil
			}

			contentLength, err := strconv.Atoi(contentLengthString)
			if err != nil {
				return 0, fmt.Errorf("invalid Content-Length header value %q: %w", contentLengthString, err)
			}

			if contentLength < 0 {
				return 0, fmt.Errorf("invalid negative Content-Length: %d", contentLength)
			}
			r.Body = make([]byte, contentLength)
			r.bodyBytesRead = 0
		}

		if r.bodyBytesRead+len(data) > len(r.Body) {
			return 0, fmt.Errorf("request body too large: %d bytes read, %d bytes remaining", r.bodyBytesRead, len(r.Body)-r.bodyBytesRead)
		}

		n := copy(r.Body[r.bodyBytesRead:], data)
		r.bodyBytesRead += n
		if r.bodyBytesRead == len(r.Body) {
			r.state = parserStateDone
		}

		return n, nil

	case parserStateDone:
		return 0, fmt.Errorf("attempting to read from a done state")
	default:
		return 0, fmt.Errorf("unknown parser state")
	}
}

func parseRequestLine(data []byte) (*RequestLine, int, error) {
	index := bytes.Index(data, []byte(CRLF))
	if index == -1 {
		return nil, 0, nil
	}

	rp, err := buildRequestLine(string(data[:index]))
	if err != nil {
		return nil, 0, err
	}

	return rp, index + len([]byte(CRLF)), nil
}

func buildRequestLine(header string) (*RequestLine, error) {
	fields := strings.Fields(header)
	if len(fields) != 3 {
		return nil, fmt.Errorf("request line does not have three components")
	}

	method, requestTarget, httpVersion := fields[0], fields[1], fields[2]

	for _, r := range method {
		if !unicode.IsUpper(r) {
			return nil, fmt.Errorf("request method can only be uppercase runes")
		}
	}

	if httpVersion != "HTTP/1.1" {
		return nil, fmt.Errorf("we only support HTTP/1.1")
	}
	httpVersion = strings.TrimPrefix(httpVersion, "HTTP/")

	return &RequestLine{HttpVersion: httpVersion, RequestTarget: requestTarget, Method: method}, nil
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	buffer := make([]byte, BUFFER_SIZE)
	readToIndex := 0
	r := &Request{state: parserStateInitialized, Headers: headers.NewHeaders()}

	for r.state != parserStateDone {
		if readToIndex >= len(buffer) {
			newBuffer := make([]byte, 2*readToIndex)
			copy(newBuffer, buffer)
			buffer = newBuffer
		}

		n, err := reader.Read(buffer[readToIndex:])
		if n > 0 {
			readToIndex += n
			for {
				consumed, parseErr := r.parse(buffer[:readToIndex])
				if parseErr != nil {
					return nil, parseErr
				}
				if consumed == 0 || r.state == parserStateDone {
					break
				}

				remaining := readToIndex - consumed
				if remaining > 0 {
					copy(buffer, buffer[consumed:readToIndex])
				}
				readToIndex = remaining
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				if r.state != parserStateDone {
					return nil, fmt.Errorf("incomplete HTTP request: connection closed unexpectedly (EOF) in state %v with %d bytes remaining in buffer: [%v]", r.state, readToIndex, buffer[:readToIndex])
				}
				break
			}
			return nil, err
		}

	}

	return r, nil
}
