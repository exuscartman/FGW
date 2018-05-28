@echo off
:: LocalSense websocket address
set addr="localhost:8080"

:: LocalSense websocket path
set path="/echo"

:: MCD push server listen port
set lport="1024"

:: DeviceID
set DevID="123"

:: ChannelID
set ChanID="1"

call FGW.exe -a %addr% -p %path% -l %lport% -d %DevID% -c %ChanID%