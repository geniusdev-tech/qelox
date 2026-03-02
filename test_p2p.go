package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	peers := []string{
		"35.188.147.227:4002",
		"34.23.115.164:4002",
		"34.145.104.213:4002",
	}

	for _, peer := range peers {
		fmt.Printf("Buscando conexão TCP com %s...\t", peer)

		d := net.Dialer{Timeout: 3 * time.Second}
		conn, err := d.Dial("tcp", peer)
		if err != nil {
			fmt.Printf("FALHA: %v\n", err)
			continue
		}

		fmt.Printf("SUCESSO (Local Addr: %s)\n", conn.LocalAddr().String())
		conn.Close()
	}
}
