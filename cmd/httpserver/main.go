package main

import (
	"crypto/sha256"
	"fmt"
	"httpfromtcp/internal/headers"
	"httpfromtcp/internal/request"
	"httpfromtcp/internal/response"
	"httpfromtcp/internal/server"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const port = 42069
const BUFFER_SIZE = 1_024
const CHUNK_SIZE = 32

const YOUR_PROBLEM_RESPONSE string = `<html>
  <head>
    <title>400 Bad Request</title>
  </head>
  <body>
    <h1>Bad Request</h1>
    <p>Your request honestly kinda sucked.</p>
  </body>
</html>`

const MY_PROBLEM_RESPONSE string = `<html>
  <head>
    <title>500 Internal Server Error</title>
  </head>
  <body>
    <h1>Internal Server Error</h1>
    <p>Okay, you know what? This one is on me.</p>
  </body>
</html>`

const SUCCESS_RESPONSE string = `<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
  </body>
</html>`

var proxyHandler server.Handler = func(w *response.Writer, req *request.Request) {
	requestTarget := strings.TrimPrefix(req.RequestLine.RequestTarget, "/httpbin")
	resp, err := http.Get(fmt.Sprintf("https://httpbin.org%s", requestTarget))
	if err != nil {
		statusCode := response.StatusCodeInternalServerError
		_ = w.WriteStatusLine(statusCode)
		_ = w.WriteHeaders(response.GetDefaultHeaders(0))
		return
	}
	defer resp.Body.Close()
	buffer := make([]byte, BUFFER_SIZE)

	if err := w.WriteStatusLine(response.StatusCodeOK); err != nil {
		log.Printf("error writing status line: %v", err)
		return
	}

	h := headers.NewHeaders()
	h.Set("Content-Type", "text/plain")
	h.Set("Transfer-Encoding", "chunked")
	h.Set("Trailer", "X-Content-Sha256, X-Content-Length")
	// h.Set("Connection", "close")
	if err := w.WriteHeaders(h); err != nil {
		log.Printf("error writing headers: %v", err)
		return
	}

	bodyLength := 0
	aggregatedBody := []byte{}
	for {
		bufferLength, err := resp.Body.Read(buffer)
		bodyLength += bufferLength
		aggregatedBody = append(aggregatedBody, buffer[:bufferLength]...)

		i := 0
		for i+CHUNK_SIZE < bufferLength {
			_, chunkErr := w.WriteChunkedBody(buffer[i : i+CHUNK_SIZE])
			if chunkErr != nil {
				log.Printf("error writing chunk: %v", chunkErr)
				return
			}
			i += CHUNK_SIZE
		}
		if bufferLength > i {
			_, chunkErr := w.WriteChunkedBody(buffer[i:bufferLength])
			if chunkErr != nil {
				log.Printf("error writing chunk: %v", chunkErr)
				return
			}
		}

		if err == io.EOF {
			_, chunkErr := w.WriteChunkedBodyDone()
			if chunkErr != nil {
				log.Printf("error writing chunk: %v", chunkErr)
			}
			break
		}
		if err != nil {
			log.Printf("error reading from body: %v", err)
			return
		}
	}

	t := headers.NewHeaders()
	t.Set("X-Content-Length", fmt.Sprintf("%d", bodyLength))
	hash := sha256.Sum256(aggregatedBody)
	t.Set("X-Content-Sha256", fmt.Sprintf("%x", hash))

	if err := w.WriteTrailers(t); err != nil {
		log.Printf("error writing trailers: %v", err)
		return
	}

}

var mainHandler server.Handler = func(w *response.Writer, req *request.Request) {
	var statusCode response.StatusCode
	var body []byte

	requestTarget := req.RequestLine.RequestTarget

	switch {
	// httpbin case handled separately
	case strings.HasPrefix(requestTarget, "/httpbin"):
		proxyHandler(w, req)
		return

	case requestTarget == "/yourproblem":
		statusCode = response.StatusCodeBadRequest
		body = []byte(YOUR_PROBLEM_RESPONSE)
	case requestTarget == "/myproblem":
		statusCode = response.StatusCodeInternalServerError
		body = []byte(MY_PROBLEM_RESPONSE)
	default:
		statusCode = response.StatusCodeOK
		body = []byte(SUCCESS_RESPONSE)
	}

	contentLength := len(body)
	headers := response.GetDefaultHeaders(contentLength)
	headers.Set("Content-Type", "text/html")

	if err := w.WriteStatusLine(statusCode); err != nil {
		log.Printf("error writing status line: %v", err)
		return
	}

	if err := w.WriteHeaders(headers); err != nil {
		log.Printf("error writing headers: %v", err)
		return
	}

	if value, exists := headers.Get("Content-Length"); !exists || value == "0" {
		return
	}

	n, err := w.WriteBody(body)
	if err != nil {
		log.Printf("error writing body: %v", err)
		return
	}
	if n != contentLength {
		log.Printf("wrote %d bytes instead of %d bytes as expected by content-length", n, contentLength)
		return
	}
}

func main() {
	log.SetFlags(log.Lshortfile)

	server, err := server.Serve(port, mainHandler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
