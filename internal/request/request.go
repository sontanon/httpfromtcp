package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

type ParserState int

const (
	ParserStateInvalid ParserState = iota
	ParserStateInitialized
	ParserStateDone
)

const BUFFER_SIZE int = 8

type Request struct {
	RequestLine RequestLine
	State       ParserState
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func (r *Request) parse(data []byte) (int, error) {
	switch r.State {
	case ParserStateInitialized:
		requestLine, n, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, nil
		}
		r.RequestLine = *requestLine
		r.State = ParserStateDone
		return n, nil
	case ParserStateDone:
		return 0, fmt.Errorf("attempting to read from a done state")
	default:
		return 0, fmt.Errorf("unknown parser state")
	}
}

const CRLF = "\r\n"

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
	r := &Request{State: ParserStateInitialized}

	for r.State != ParserStateDone {
		if readToIndex >= len(buffer) {
			newBuffer := make([]byte, 2*readToIndex)
			copy(newBuffer, buffer)
			buffer = newBuffer
		}

		n, err := reader.Read(buffer[readToIndex:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				r.State = ParserStateDone
				break
			}
			return nil, err
		}

		readToIndex += n
		m, err := r.parse(buffer[:readToIndex])
		if err != nil {
			return nil, err
		}
		if m != 0 {
			newBuffer := make([]byte, len(buffer)-m)
			copy(newBuffer, buffer[m:])
			buffer = newBuffer
			readToIndex -= m
		}
	}

	return r, nil
}
