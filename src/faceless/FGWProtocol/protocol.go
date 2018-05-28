package FGWProtocol

import (
	"encoding/json"
	"strings"
	"io"
	"log"
	"encoding/binary"
	"time"
	"bytes"
	"github.com/axgle/mahonia"
	"faceless/Misc"
)

const (
	frameHead = 0xCC5F
	frameTail = 0xAABB
)

const (
	frameType = 0x03
	alarmType = 0x01
	correlationId = 0x0011
	alarmTimestamp = 1431573999604
	alarmInfo = "进入区域报警，电子围栏名称：禁区1,联动摄像头"
)

const (
	Origin = 0  // value --> 0
	FindHead = 1
	NeedTail = 2
	MatchTail = 3
)

type HeartBeat struct {
	MsgType string `json: MsgType`
	State string `json: State`
}

type Alarm struct {
	DeviceID string `json: DeviceID`
	ChanelID string `json: ChanelID`
	MsgType string `json: MsgType`
	CID string `json: CID`
	DeviceIP string `json: DeviceIP`
	DeviceMAC string `json: DeviceMAC`
	Port string `json: Port`
	UserName string `json: UserName`
	PassWord string `json: PassWord`
	Time string `json: Time`
	ExtInfo string `json: ExtInfo`
}

func UnpackLS(p []byte) []byte {
	return []byte{}
}

func DecodeHeartBeat(jsonStream string) {
	dec := json.NewDecoder(strings.NewReader(jsonStream))
	for {
		var m HeartBeat
		err := dec.Decode(&m)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Println(err)
		} else {
			log.Printf("%s: %s\n", m.MsgType, m.State)
		}
	}
}

func EncodeHeartBeat(s string) []byte {
	var hb HeartBeat
	hb.MsgType = "HeartBeat"
	hb.State = s
	hbbs, _ := json.Marshal(hb)
	return hbbs
}

func UnPack(buffer []byte) ([][]byte, []byte) {
	ret := make([][]byte, 0)
	flag := Origin
	length := len(buffer)
	message := make([]byte, length)
	copy(message, buffer)
	remainedBuf := make([]byte, 0)
	for {
		var msgEnd = 0
		flag, message, length = GetFrameHead(message, length)
		if (flag == FindHead && length > 2) {
			flag, msgEnd = GetFrameTail(message, length)
			if (flag == MatchTail) {
				ret = append(ret, message[2:msgEnd])
				length = length - msgEnd - 2
				if length > 0 {
					message = message[msgEnd+2:]
					continue
				} else {
					remainedBuf = make([]byte, 0)
					break
				}
			} else {
				remainedBuf = message
				break
			}
		} else {
			remainedBuf = message
			break
		}
		remainedBuf = message
	}
	return ret, remainedBuf
}

func GetFrameHead(buffer []byte, length int) (int,[]byte, int) {
	var i int
	flag := Origin
	for i = 0; i < length; i++ {
		if length < i + 2 {
			break
		}
		tempPacketData := new(bytes.Buffer)
		binary.Write(tempPacketData, binary.BigEndian, uint16(frameHead))
		if bytes.Equal(tempPacketData.Bytes(), buffer[i:i+2]) {
			flag = FindHead
			break
		}
	}
	if i == length {
		return flag, make([]byte, 0), length-i
	}
	return flag, buffer[i:], length-i
}


func GetFrameTail(buffer []byte, length int) (int, int){
	var i int
	flag := NeedTail
	for i = 2; i < length; i++ {
		if length < i + 2 {
			break
		}
		tempPacketData := new(bytes.Buffer)
		binary.Write(tempPacketData, binary.BigEndian, uint16(frameTail))
		if bytes.Equal(tempPacketData.Bytes(), buffer[i:i+2]) {
			flag = MatchTail
			break
		}
	}
	if flag == NeedTail {
		i = 0
	}
	return flag, i
}

func TransLS2MCD(devId string, chanId string, src []byte) []byte {
	jsonStr := make([]byte, 0)
	// check crc
	checkCrc := Misc.UsMBCRC16(src[:len(src)-2], len(src)-2)
	var crcSrc uint16
	binary.Read(bytes.NewBuffer(src[len(src)-2:]), binary.BigEndian, &crcSrc)
	if uint16(checkCrc) != crcSrc {
		log.Println("CRC check failed: ", src)
		return jsonStr
	}
	switch src[0] {
	case 0x03:
		var jsonAlarm Alarm
		// 过滤报警类型
		switch src[1] {
		case 0x01:
			jsonAlarm.DeviceID = devId
			jsonAlarm.ChanelID = chanId
			jsonAlarm.MsgType = "Alarm"
			jsonAlarm.CID = "1000"
			jsonAlarm.DeviceIP = ""
			jsonAlarm.DeviceMAC = ""
			jsonAlarm.Port = ""
			jsonAlarm.UserName = ""
			jsonAlarm.PassWord = ""
			jsonAlarm.ExtInfo = "{}"
			var timestamp int64
			binary.Read(bytes.NewBuffer(src[4:12]), binary.BigEndian, &timestamp)
			tm := time.Unix(timestamp/1000, 0)
			jsonAlarm.Time = tm.Format("2006-01-02 15:04:05")
			jsonStr,_ = json.Marshal(jsonAlarm)
		default:
		}
	default:
	}
	return jsonStr
}

//BytesCombine 多个[]byte数组合并成一个[]byte
func BytesCombine(pBytes ...[]byte) []byte {
	return bytes.Join(pBytes, []byte(""))
}

func MakeFakePacket() []byte {
	alarmInfoGbk := []byte(mahonia.NewEncoder("gbk").ConvertString(alarmInfo))
	alarmInfoByte := make([]byte, 120)
	copy(alarmInfoByte, alarmInfoGbk)
	tempPacketData := new(bytes.Buffer)
	binary.Write(tempPacketData, binary.BigEndian, uint16(frameHead))
	binary.Write(tempPacketData, binary.BigEndian, uint8(frameType))
	binary.Write(tempPacketData, binary.BigEndian, uint8(alarmType))
	binary.Write(tempPacketData, binary.BigEndian, uint16(correlationId))

	/*
	toBeChange := time.Now()
	log.Println(" ", toBeChange)
	ts := strconv.FormatInt(toBeChange.UnixNano()/1000000, 10)
	log.Println(ts)
	log.Println(" ", toBeChange.Format("2006-01-02 15:04:05"))

	loc, _ := time.LoadLocation("Local")
	theTime, _ := time.ParseInLocation("2006-01-02 15:04:05.000", "2015-05-14 11:26:39.604", loc)
	log.Println(" ", theTime)
	ts1 := strconv.FormatInt(theTime.UnixNano()/1000000, 10)
	log.Println(" ", ts1)
	*/
	binary.Write(tempPacketData, binary.BigEndian, uint64(time.Now().UnixNano()/1000000))
	tempPacketData.Write(alarmInfoByte)

	crc16Code := Misc.UsMBCRC16(tempPacketData.Bytes()[2:], tempPacketData.Len()-2 )
	crc16CodeBuffer := new(bytes.Buffer)
	binary.Write(crc16CodeBuffer, binary.BigEndian, uint16(crc16Code))
	frameTailBuffer := new(bytes.Buffer)
	binary.Write(frameTailBuffer, binary.BigEndian, uint16(frameTail))

	packetData := BytesCombine(tempPacketData.Bytes(), crc16CodeBuffer.Bytes(), frameTailBuffer.Bytes())

	return packetData
}