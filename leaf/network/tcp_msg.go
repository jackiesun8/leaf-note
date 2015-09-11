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
	lenMsgLen    int    //消息长度占用字节数
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
	var b [4]byte                //先声明一个4字节切片
	bufMsgLen := b[:p.lenMsgLen] //根据消息长度占用字节数重新取得对应长度的切片

	// read len
	if _, err := io.ReadFull(conn, bufMsgLen); err != nil { //读取消息长度
		return nil, err
	}

	// parse len
	//解析长度
	var msgLen uint32
	switch p.lenMsgLen {
	case 1:
		msgLen = uint32(bufMsgLen[0]) //长度一个字节
	case 2:
		if p.littleEndian {
			msgLen = uint32(binary.LittleEndian.Uint16(bufMsgLen)) //长度两个字节小端
		} else {
			msgLen = uint32(binary.BigEndian.Uint16(bufMsgLen)) //长度两个字节大端
		}
	case 4:
		if p.littleEndian {
			msgLen = binary.LittleEndian.Uint32(bufMsgLen) //长度四个字节小端
		} else {
			msgLen = binary.BigEndian.Uint32(bufMsgLen) //长度四个字节大端
		}
	}

	// check len
	//检查长度
	if msgLen > p.maxMsgLen { //超过了最大长度
		return nil, errors.New("message too long")
	} else if msgLen < p.minMsgLen { //小于最小长度
		return nil, errors.New("message too short")
	}

	// data
	msgData := make([]byte, msgLen)                       //创建对应长度的字节切片
	if _, err := io.ReadFull(conn, msgData); err != nil { //读取数据
		return nil, err
	}

	return msgData, nil //返回读取的数据
}

// goroutine safe
//发送消息
func (p *MsgParser) Write(conn *TCPConn, args ...[]byte) error { //传入多个字节切片
	// get len
	//计算长度
	var msgLen uint32
	for i := 0; i < len(args); i++ { //遍历所有字节切片
		msgLen += uint32(len(args[i])) //累加长度
	}

	// check len
	// 检查长度
	if msgLen > p.maxMsgLen { //超过了最大长度
		return errors.New("message too long")
	} else if msgLen < p.minMsgLen { //小于最小长度
		return errors.New("message too short")
	}

	msg := make([]byte, uint32(p.lenMsgLen)+msgLen) //创建len+len(data)长度的字节切片

	// write len
	//写入长度
	switch p.lenMsgLen {
	case 1:
		msg[0] = byte(msgLen) //一个字节
	case 2:
		if p.littleEndian {
			binary.LittleEndian.PutUint16(msg, uint16(msgLen)) //两个字节小端
		} else {
			binary.BigEndian.PutUint16(msg, uint16(msgLen)) //两个字节大端
		}
	case 4:
		if p.littleEndian {
			binary.LittleEndian.PutUint32(msg, msgLen) //四个字节小端
		} else {
			binary.BigEndian.PutUint32(msg, msgLen) //四个字节大端
		}
	}

	// write data
	//写入数据
	l := p.lenMsgLen
	for i := 0; i < len(args); i++ { //遍历所有字节切片
		copy(msg[l:], args[i]) //拷贝数据
		l += len(args[i])      //游标
	}

	conn.Write(msg) //发送数据

	return nil
}
