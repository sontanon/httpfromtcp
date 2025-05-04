package headers

import (
	"bytes"
	"fmt"
	"strings"
)

type Headers map[string]string

const CRLF = "\r\n"
const COLON = ":"

func NewHeaders() Headers {
	return make(Headers)
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	index := bytes.Index(data, []byte(CRLF))
	if index == -1 {
		return 0, false, nil
	}

	if index == 0 {
		return 0, true, nil
	}

	key, value, found := strings.Cut(string(data[:index]), COLON)
	if !found {
		return 0, false, fmt.Errorf("invalid header does not contain colon separator")
	}

	if key != strings.TrimRight(key, " \t\n\r") {
		return 0, false, fmt.Errorf("invalid header key contains trailing whitespace")
	}

	h[strings.TrimSpace(key)] = strings.TrimSpace(value)

	return index + len([]byte(CRLF)), false, nil
}
