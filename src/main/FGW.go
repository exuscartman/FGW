// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"
	"net"
	"runtime"

	"github.com/gorilla/websocket"
	"faceless/FGWProtocol"
)

// quanju
var addr = flag.String("addr", "localhost:8080", "http service address")
var path = flag.String("path", "/echo", "websocket handler path")
var listen = flag.String("listen", "0.0.0.0:1024", "listen port")
var devId = flag.String("dev", "123", "device ID")
var chanId = flag.String("chan" , "1", "channel ID")

var alarmQueue chan []byte
var heartbeatQueue chan []byte

// MCD客户端是否连接， 共享变量有一个读通道和一个写通道组成
type IsConnected struct {
	get chan int
	set chan int
}

var connWD IsConnected

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	log.SetFlags(0)
	// 设置缓冲深度100的队列
	alarmQueue = make(chan []byte, 100)
	heartbeatQueue = make(chan []byte, 10)
	connWD = IsConnected{make(chan int), make(chan int)}
	IsConnectedWatchdog(connWD)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})

	go localSenseCli(done, interrupt)
	go MCDServer(done)

	for {
		select {
		case <-done:
			return
		}
	}
}

//共享变量维护协程
func IsConnectedWatchdog(v IsConnected) {
	go func() {
		//初始值, 0 unconnected, 1 connected
		var value int = 0
		for {
			//监听读写通道，完成服务
			select {
			case value = <-v.set:
			case v.get <- value:
			}
		}
	}()
}

func MCDServer(done chan struct{}){
	defer close(done)

	netListen, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Println("Fatal error: ", err.Error())
		return
	}
	defer netListen.Close()

	for {
		log.Println("Waiting for clients")
		conn, err := netListen.Accept()
		if err != nil {
			continue
		}
		log.Println(conn.RemoteAddr().String(), " tcp connect sucess")
		// 开始预警队列的数据接收
		connWD.set <- 1
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buffer := make([]byte, 2048)
	for {
		// 设置100微妙读取超时
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
		n, err := conn.Read(buffer)
		if err != nil {
			if nerr, ok := err.(net.Error); !ok || !nerr.Timeout() {
				log.Println(conn.RemoteAddr().String(), " connection error: ", err)
				connWD.set <- 0
				return
			}
		}
		// 收到任何消息都认为时心跳请求，直接回复心跳应答
		if n > 0 {
			heartbeatQueue <- FGWProtocol.EncodeHeartBeat("ACK")
		}
		select {
		case m := <- alarmQueue:
			alarmMsg := FGWProtocol.TransLS2MCD(*devId, *chanId, m)
			if len(alarmMsg) == 0 {
				// log.Printf("dump: [%X] \n", m)
				continue
			}
			_, err := conn.Write(alarmMsg)
			if err != nil {
				log.Println("write:", err)
				connWD.set <- 0
				return
			}
		case hb := <- heartbeatQueue:
			_, err := conn.Write(hb)
			if err != nil {
				log.Println("write:", err)
				connWD.set <- 0
				return
			}
		case <-time.After(time.Millisecond * 100):
		}
	}
}

func localSenseCli(done chan struct{}, interrupt chan os.Signal) {
	// 线程退出时代理网关程序终止，未作自动重连
	defer close(done)

	u := url.URL{Scheme: "ws", Host: *addr, Path: *path }
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	// 接收推送结束标志
	LSCDone := make(chan struct{})

	// 接收推送消息
	go func() {
		defer close(LSCDone)

		tmpBuffer := make([]byte, 0)

		for {
			fs := make([][]byte, 0)
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			// log.Printf("recv: [%X] \n", message)
			tmpBuffer = append(tmpBuffer, message...)
			fs, tmpBuffer = FGWProtocol.UnPack(tmpBuffer)
			numFrames := len(fs)
			// 处理每个数据包
			var j int
			for j = 0; j < numFrames; j++ {
				// 是否有MCD客户端连接
				isconn := <-connWD.get
				if (isconn == 1) {
					alarmQueue <- fs[j]
				} else {
					log.Printf("dump: [%X] \n", message)
				}
			}
		}
	}()

	for {
		select {
		case <-LSCDone:
			return
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-LSCDone:
			case <-time.After(time.Second):
			}
			return
		}
	}
}