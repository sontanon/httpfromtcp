package headers

import (
	"bytes"
	"fmt"
	"strings"
)

type Headers map[string]string

const CRLF = "\r\n"
const CRLF_LENGTH = 2
const COLON = ":"

const STANDARD_RUNES = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!#$%&'*+-.^_`|~"

func NewHeaders() Headers {
	return make(Headers)
}

func invalidRune(r rune) bool {
	return !strings.ContainsRune(STANDARD_RUNES, r)
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	index := bytes.Index(data, []byte(CRLF))
	if index == -1 {
		return 0, false, nil
	}

	if index == 0 {
		return CRLF_LENGTH, true, nil
	}

	key, value, found := strings.Cut(string(data[:index]), COLON)
	if !found {
		return 0, false, fmt.Errorf("invalid header does not contain colon separator")
	}

	if key != strings.TrimRight(key, " \t\n\r") {
		return 0, false, fmt.Errorf("invalid header key contains trailing whitespace")
	}

	key = strings.TrimSpace(key)
	if strings.ContainsFunc(key, invalidRune) {
		return 0, false, fmt.Errorf("invalid header key contains invalid characters")
	}

	key = strings.ToLower(key)

	_, exists := h[key]
	if !exists {
		h[key] = strings.TrimSpace(value)
	} else {
		h[key] = fmt.Sprintf("%s, %s", h[key], strings.TrimSpace(value))
	}

	return index + CRLF_LENGTH, false, nil
}

func (h Headers) Get(key string) (string, bool) {
	value, ok := h[strings.ToLower(key)]
	return value, ok
}
