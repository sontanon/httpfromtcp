package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

func getLinesChannel(f io.ReadCloser) <-chan string {
	c := make(chan string)

	go func() {
		buffer := make([]byte, 8)
		var currentLine string

		defer f.Close()
		defer close(c)

		for {
			n, err := f.Read(buffer)
			if err != nil && err != io.EOF {
				log.Fatalf("failed during buffered read: %v", err)
			}

			parts := strings.Split(string(buffer[:n]), "\n")
			for _, part := range parts[:len(parts)-1] {
				currentLine += part
				c <- currentLine
				currentLine = ""
			}

			currentLine += parts[len(parts)-1]
			if err == io.EOF {
				break
			}
		}

		if currentLine != "" {
			c <- currentLine
		}
	}()

	return c
}

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

		linesChannel := getLinesChannel(conn)
		for line := range linesChannel {
			fmt.Printf("read: %s\n", line)
		}
		log.Println("channel is closed")
	}
}
