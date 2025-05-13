package server

import (
	"fmt"
	"httpfromtcp/internal/request"
	"httpfromtcp/internal/response"
	"log"
	"net"
	"sync/atomic"
)

const BUFFER_SIZE = 1_024

type Server struct {
	listener net.Listener
	handler  Handler
	open     *atomic.Bool
}

type HandlerError struct {
	Status  response.StatusCode
	Message string
}

type Handler func(w *response.Writer, req *request.Request)

func (he HandlerError) WriteError(w *response.Writer) {
	body := []byte(he.Message)
	contentLength := len(body)
	headers := response.GetDefaultHeaders(contentLength)

	_ = w.WriteStatusLine(he.Status)
	_ = w.WriteHeaders(headers)
	_, _ = w.WriteBody(body)
}

func Serve(port int, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	open := atomic.Bool{}
	open.Store(true)

	server := Server{
		listener: listener,
		handler:  handler,
		open:     &open,
	}

	go server.listen()

	return &server, nil
}

func (s *Server) Close() error {
	if err := s.listener.Close(); err != nil {
		return err
	}
	s.open.Store(false)
	return nil
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil && s.open.Load() {
			log.Printf("failed to establish a connection: %v", err)
		}

		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	writer := response.NewWriter(conn)

	req, err := request.RequestFromReader(conn)
	if err != nil {
		HandlerError{
			Status:  response.StatusCodeBadRequest,
			Message: err.Error(),
		}.WriteError(&writer)
		return
	}

	s.handler(&writer, req)
}
