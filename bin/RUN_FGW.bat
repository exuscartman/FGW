@echo off
:: LocalSense websocket address
:: ���з������ĵ�ַ�Ͷ˿�
set addr="localhost:8090"

:: LocalSense websocket path
:: ����Websocket�ӿ�
set path="/echo"

:: MCD push server listen port
:: ���մ󻪿ͻ��˵������˿�
set lport="1024"

:: DeviceID
:: ����Ϣ���е�DeviceID
:: set DevID="123"

:: ChannelID
:: ����Ϣ���е�ChannelID
set ChanID="1"

call FGW.exe -a %addr% -p %path% -l %lport%  -c %ChanID%