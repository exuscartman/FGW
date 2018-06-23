package main

import (
	"net"
	"fmt"
	"os"
	"os/signal"
	"time"
	"faceless/FGWProtocol"
	"encoding/hex"
	"faceless/Misc"
)

func main() {
	// crc test
	var (
		strHex = "010C1E5A00000006FFFFFFD9000001021003A72E65040129580000000600000065000001041003A72162040164B6000000000000003C000001031003A7216204018A7F0000000600000011000001031003A72D3304019EF00000000000000003000001011003A7349A0401A5FA0000000000000065000001041003A7373A0401A62000000006FFFFFFBD000001031003A714AE0401EE6200000006FFFFFFF6000001051003A727F00401EE9B0000000000000003000001051003A72A820401EEBC00000006FFFFFFD9000001051003A731040401EEE400000006FFFFFFD9000001051003A72D330401F06C00000000FFFFFFF6000001051003A732360401"
		crcSrc = "4094"
	)
	// src := []byte("abc")
	// encodeStr := hex.EncodeToString(src)
	inputBytes, _ := hex.DecodeString(strHex)
	crc16Code := Misc.UsMBCRC16(inputBytes, len(inputBytes))
	fmt.Printf("%X\n", crc16Code)
	fmt.Println(crcSrc)

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
