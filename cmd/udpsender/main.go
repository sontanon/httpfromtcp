package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	// Resolve the address localhost:42069
	raddr, err := net.ResolveUDPAddr("udp", ":42069")
	if err != nil {
		log.Fatal("failed to resolve UDP address")
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		log.Fatal("failed to prepare UDP connection")
	}
	defer conn.Close()

	buffer := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(">")
		line, err := buffer.ReadString('\n')
		if err != nil {
			log.Printf("failed to read from stdin: %v", err)
		}

		_, err = conn.Write([]byte(line))
		if err != nil {
			log.Printf("failed to write to UDP connection: %v", err)
		}
	}
}
