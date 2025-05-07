package server

import (
	"fmt"
	"httpfromtcp/internal/request"
	"httpfromtcp/internal/response"
	"log"
	"net"
	"sync/atomic"
)

type Server struct {
	listener net.Listener
	open     *atomic.Bool
}

func Serve(port int) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	open := atomic.Bool{}
	open.Store(true)

	server := Server{listener: listener, open: &open}

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

	r, err := request.RequestFromReader(conn)
	if err != nil {
		log.Printf("error processing request: %v", err)
		return
	}
	log.Println(r.PrettyPrint())

	if err := response.WriteStatusLine(conn, response.StatusCodeOK); err != nil {
		log.Printf("failed writing status line: %v", err)
		return
	}
	h := response.GetDefaultHeaders(0)
	if err := response.WriteHeaders(conn, h); err != nil {
		log.Printf("failed writing headers: %v", err)
		return
	}
}
