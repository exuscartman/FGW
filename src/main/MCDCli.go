package main

import (
	"net"
	"fmt"
	"os"
	"os/signal"
	"time"
	"faceless/FGWProtocol"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	server := "127.0.0.1:1024"
	tcpAddr, err := net.ResolveTCPAddr("tcp4", server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("connect success")

	done := make(chan struct{})

	ticker := time.NewTicker(3*time.Second)
	defer ticker.Stop()

	go func() {
		defer close(done)
		for {
			buffer := make([]byte, 1024)
			n, err := conn.Read(buffer)
			if err != nil {
				fmt.Println("read:", err)
				return
			}
			fmt.Printf("recv:%s \n", string(buffer[:n]))
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			_, err := conn.Write(FGWProtocol.EncodeHeartBeat("REQ"))
			if err != nil {
				fmt.Println("write:", err)
				return
			}
		case <-interrupt:
			return
		}

	}

}
