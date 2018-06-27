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
	"net/http"
	"encoding/json"

	"github.com/gorilla/websocket"
	"faceless/FGWProtocol"
	l4g "github.com/alecthomas/log4go"
	"sync"
)

//import loggerNet "github.com/alecthomas/log4go"

// quanju
var addr = flag.String("a", "localhost:9001", "http service address")
var path = flag.String("p", "/", "websocket handler path")
var lport = flag.String("l", "1024", "listen port")
// var devId = flag.String("d", "123", "device ID")
var chanId = flag.String("c" , "1", "channel ID")
var userName = flag.String("u", "admin", "websocket login user name")
var password = flag.String("s", "localsense", "websocket login password")
var subProtocol = flag.String("t", "localSensePush-protocol", "websocket subProtocol")


var alarmQueue chan []byte
var heartbeatQueue chan []byte

// MCD客户端是否连接， 共享变量有一个读通道和一个写通道组成
type IsConnected struct {
	get chan int
	set chan int
}

var connWD IsConnected

var noAlarmPeriodMap sync.Map

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	log.SetFlags(0)

	loginPacket := FGWProtocol.MakeLoginPacket(*userName, *password)

	l4g.LoadConfiguration("FGW.logcfg.xml")
	l4g.Finest("login packet: [%X]", loginPacket)

	// 设置缓冲深度100的队列
	alarmQueue = make(chan []byte, 100)
	heartbeatQueue = make(chan []byte, 10)
	connWD = IsConnected{make(chan int), make(chan int)}
	IsConnectedWatchdog(connWD)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})

	go localSenseCli(done, interrupt, loginPacket)
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

	netListen, err := net.Listen("tcp", "0.0.0.0:" + *lport)
	if err != nil {
		l4g.Error("Fatal error: %s", err.Error())
		//log.Println("Fatal error: ", err.Error())
		return
	}
	defer netListen.Close()

	//log.Println("Waiting for clients")
	l4g.Info("Waiting for clients")

	for {
		conn, err := netListen.Accept()
		if err != nil {
			l4g.Warn("accept: %s", err)
			continue
		}
		l4g.Info("%s tcp connect sucess", conn.RemoteAddr().String())
		//log.Println(conn.RemoteAddr().String(), " tcp connect sucess")
		// 开始预警队列的数据接收
		connWD.set <- 1
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	go func() {
		for {
			select {
			case <- time.After(time.Second*20):
				noAlarmPeriodMap.Range(func(k, v interface{}) bool {
					noAlarmPeriodMap.Delete(k)
					return true
				})
			}
		}
	} ()

	buffer := make([]byte, 2048)
	for {
		// 设置100微秒读取超时
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
		n, err := conn.Read(buffer)
		if err != nil {
			if nerr, ok := err.(net.Error); !ok || !nerr.Timeout() {
				l4g.Warn("%s connection error: %s", conn.RemoteAddr().String(), err)
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
			alarmMsg := FGWProtocol.TransLS2MCD(*chanId, m)
			if len(alarmMsg) == 0 {
				// log.Printf("dump: [%X] \n", m)
				l4g.Finest("filtered: [%X]", m)
				continue
			}
			var dat map[string]string
			err := json.Unmarshal([]byte(alarmMsg), &dat)
			_, isSent := noAlarmPeriodMap.LoadOrStore(dat["CID"]+dat["DeviceID"], true)
			if isSent {
				l4g.Finest("Repeated: [%s]", alarmMsg)
				continue
			}
			l4g.Finest("send: [%s]", alarmMsg)
			_, err = conn.Write(alarmMsg)
			if err != nil {
				l4g.Warn("write: %s", err)
				//log.Println("write:", err)
				connWD.set <- 0
				return
			}
		case hb := <- heartbeatQueue:
			_, err := conn.Write(hb)
			if err != nil {
				l4g.Warn("write: %s", err)
				connWD.set <- 0
				return
			}
		case <-time.After(time.Millisecond * 100):
		}
	}
}

func localSenseCli(done chan struct{}, interrupt chan os.Signal, loginPacket []byte) {
	// 线程退出时代理网关程序终止，未作自动重连
	defer close(done)

	u := url.URL{Scheme: "ws", Host: *addr, Path: *path }
	l4g.Info("connecting to %s", u.String())
	//log.Printf("connecting to %s", u.String())

	var header = make(http.Header)
	header.Add("Sec-WebSocket-Protocol",*subProtocol)
	header.Add("Origin", "file://")

	c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		l4g.Error("dial: %s", err)
		return
	}
	defer c.Close()
	l4g.Info("connect ws successfully")

	// 接收推送结束标志
	LSCDone := make(chan struct{})

	// 登录
	err = c.WriteMessage(websocket.BinaryMessage, loginPacket)
	if err != nil {
		l4g.Error("login failed: ", err)
		return
	}
	l4g.Info("login successfully")

	// 接收推送消息
	go func() {
		defer close(LSCDone)

		tmpBuffer := make([]byte, 0)
		for {
			fs := make([][]byte, 0)
			_, message, err := c.ReadMessage()
			if err != nil {
				l4g.Warn("read: %s", err)
				//log.Println("read:", err)
				return
			}
			// log.Printf("recv: [%X] \n", message)
			l4g.Finest("recv: [%X]", message)
			fs, tmpBuffer = FGWProtocol.UnPackLS(append(tmpBuffer, message...))
			// 处理每个数据包
			for j := 0; j < len(fs); j++ {
				// 是否有MCD客户端连接
				if (<-connWD.get == 1) {
					l4g.Finest("unpack: [%X]", fs[j])
					alarmQueue <- fs[j]
				} else {
					l4g.Finest("dump: [%X]", fs[j])
				}
			}
		}
	}()

	for {
		select {
		case <-LSCDone:
			return
		case <-interrupt:
			l4g.Info("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				l4g.Warn("write close: %s", err)
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