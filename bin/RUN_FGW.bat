@echo off
:: LocalSense websocket address
:: ���з������ĵ�ַ�Ͷ˿�
set addr="localhost:9001"

:: LocalSense websocket path
:: ����Websocket�ӿ�
set path="/"

:: MCD push server listen port
:: ���մ󻪿ͻ��˵������˿�
set lport="1024"

:: LocalSense websocket user name
:: ���е�¼�û���
set user="admin"

:: LocalSense websocket password
:: ���е�¼����
set pwd="localsense"

:: DeviceID
:: ����Ϣ���е�DeviceID
:: set DevID="123"

:: ChannelID
:: ����Ϣ���е�ChannelID
set ChanID="1"

call FGW.exe -a %addr% -p %path% -l %lport% -c %ChanID% -u %user% -s %pwd%