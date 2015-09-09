package network

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// --------------
// | len | data |
// --------------
//消息解析器类型定义
type MsgParser struct {
	lenMsgLen    int    //消息长度的长度(上图len)
	minMsgLen    uint32 //最小消息长度
	maxMsgLen    uint32 //最大消息长度
	littleEndian bool   //是否是小端
}

//创建消息解析器
func NewMsgParser() *MsgParser {
	p := new(MsgParser)    //创建消息解析器
	p.lenMsgLen = 2        //长度默认为2
	p.minMsgLen = 1        //最小长度默认为1
	p.maxMsgLen = 4096     //最大长度默认为4096
	p.littleEndian = false //默认为大端

	return p
}

// It's dangerous to call the method on reading or writing
//设置消息长度
func (p *MsgParser) SetMsgLen(lenMsgLen int, minMsgLen uint32, maxMsgLen uint32) {
	if lenMsgLen == 1 || lenMsgLen == 2 || lenMsgLen == 4 { //消息长度的有效长度为1，2，4
		p.lenMsgLen = lenMsgLen
	}
	//最小，最大消息长度不为0即可
	if minMsgLen != 0 {
		p.minMsgLen = minMsgLen
	}
	if maxMsgLen != 0 {
		p.maxMsgLen = maxMsgLen
	}
	//根据消息长度的长度计算data最大长度，并检查最小，最大消息长度正确性，如不正确，则重置
	var max uint32
	switch p.lenMsgLen {
	case 1:
		max = math.MaxUint8
	case 2:
		max = math.MaxUint16
	case 4:
		max = math.MaxUint32
	}
	if p.minMsgLen > max { //最小长度不应超过实际最大值
		p.minMsgLen = max
	}
	if p.maxMsgLen > max { //最大长度不应超过实际最大值
		p.maxMsgLen = max
	}
}

// It's dangerous to call the method on reading or writing
//设置字节序，true小端，false大端
func (p *MsgParser) SetByteOrder(littleEndian bool) {
	p.littleEndian = littleEndian
}

// goroutine safe
//读取消息
func (p *MsgParser) Read(conn *TCPConn) ([]byte, error) {
	var b [4]byte
	bufMsgLen := b[:p.lenMsgLen]

	// read len
	if _, err := io.ReadFull(conn, bufMsgLen); err != nil {
		return nil, err
	}

	// parse len
	var msgLen uint32
	switch p.lenMsgLen {
	case 1:
		msgLen = uint32(bufMsgLen[0])
	case 2:
		if p.littleEndian {
			msgLen = uint32(binary.LittleEndian.Uint16(bufMsgLen))
		} else {
			msgLen = uint32(binary.BigEndian.Uint16(bufMsgLen))
		}
	case 4:
		if p.littleEndian {
			msgLen = binary.LittleEndian.Uint32(bufMsgLen)
		} else {
			msgLen = binary.BigEndian.Uint32(bufMsgLen)
		}
	}

	// check len
	if msgLen > p.maxMsgLen {
		return nil, errors.New("message too long")
	} else if msgLen < p.minMsgLen {
		return nil, errors.New("message too short")
	}

	// data
	msgData := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, msgData); err != nil {
		return nil, err
	}

	return msgData, nil
}

// goroutine safe
//发送消息
func (p *MsgParser) Write(conn *TCPConn, args ...[]byte) error {
	// get len
	var msgLen uint32
	for i := 0; i < len(args); i++ {
		msgLen += uint32(len(args[i]))
	}

	// check len
	if msgLen > p.maxMsgLen {
		return errors.New("message too long")
	} else if msgLen < p.minMsgLen {
		return errors.New("message too short")
	}

	msg := make([]byte, uint32(p.lenMsgLen)+msgLen)

	// write len
	switch p.lenMsgLen {
	case 1:
		msg[0] = byte(msgLen)
	case 2:
		if p.littleEndian {
			binary.LittleEndian.PutUint16(msg, uint16(msgLen))
		} else {
			binary.BigEndian.PutUint16(msg, uint16(msgLen))
		}
	case 4:
		if p.littleEndian {
			binary.LittleEndian.PutUint32(msg, msgLen)
		} else {
			binary.BigEndian.PutUint32(msg, msgLen)
		}
	}

	// write data
	l := p.lenMsgLen
	for i := 0; i < len(args); i++ {
		copy(msg[l:], args[i])
		l += len(args[i])
	}

	conn.Write(msg)

	return nil
}
