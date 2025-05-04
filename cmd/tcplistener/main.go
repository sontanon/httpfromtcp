package main

import (
	"fmt"
	"httpfromtcp/internal/request"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Fatal("failed to setup listener")
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("failed to establish a connection")
		}
		log.Println("connection accepted")

		r, err := request.RequestFromReader(conn)
		if err != nil {
			log.Fatal("failed to create request from connection")
		}
		fmt.Println(r.PrettyPrint())
	}
}
