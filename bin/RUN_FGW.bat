@echo off
:: LocalSense websocket address
:: 清研服务器的地址和端口
set addr="localhost:8090"

:: LocalSense websocket path
:: 清研Websocket接口
set path="/echo"

:: MCD push server listen port
:: 接收大华客户端的侦听端口
set lport="1024"

:: DeviceID
:: 大华消息体中的DeviceID
:: set DevID="123"

:: ChannelID
:: 大华消息体中的ChannelID
set ChanID="1"

call FGW.exe -a %addr% -p %path% -l %lport% -c %ChanID%