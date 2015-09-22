package protobuf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/name5566/leaf/chanrpc"
	"github.com/name5566/leaf/log"
	"math"
	"reflect"
)

// -------------------------
// | id | protobuf message |
// -------------------------
//处理器类型定义
type Processor struct {
	littleEndian bool                    //是否是小端
	msgInfo      []*MsgInfo              //消息信息切片 JSON的是用映射存，protobuf的是用切片存
	msgID        map[reflect.Type]uint16 //消息ID映射
}

//消息信息类型定义
type MsgInfo struct {
	msgType    reflect.Type    //消息类型
	msgRouter  *chanrpc.Server //处理消息的RPC服务器
	msgHandler MsgHandler      //消息处理函数
}

//消息处理函数定义
type MsgHandler func([]interface{})

//创建一个处理器
func NewProcessor() *Processor {
	p := new(Processor)                     //创建处理器
	p.littleEndian = false                  //默认大端
	p.msgID = make(map[reflect.Type]uint16) //创建映射
	return p
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetByteOrder(littleEndian bool) {
	p.littleEndian = littleEndian
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) Register(msg proto.Message) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("protobuf message pointer required")
	}
	if _, ok := p.msgID[msgType]; ok {
		log.Fatal("message %s is already registered", msgType)
	}
	if len(p.msgInfo) >= math.MaxUint16 {
		log.Fatal("too many protobuf messages (max = %v)", math.MaxUint16)
	}

	i := new(MsgInfo)
	i.msgType = msgType
	p.msgInfo = append(p.msgInfo, i)
	p.msgID[msgType] = uint16(len(p.msgInfo) - 1)
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetRouter(msg proto.Message, msgRouter *chanrpc.Server) {
	msgType := reflect.TypeOf(msg)
	id, ok := p.msgID[msgType]
	if !ok {
		log.Fatal("message %s not registered", msgType)
	}

	p.msgInfo[id].msgRouter = msgRouter
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetHandler(msg proto.Message, msgHandler MsgHandler) {
	msgType := reflect.TypeOf(msg)
	id, ok := p.msgID[msgType]
	if !ok {
		log.Fatal("message %s not registered", msgType)
	}

	p.msgInfo[id].msgHandler = msgHandler
}

// goroutine safe
func (p *Processor) Route(msg proto.Message, userData interface{}) error {
	msgType := reflect.TypeOf(msg)
	id, ok := p.msgID[msgType]
	if !ok {
		return fmt.Errorf("message %s not registered", msgType)
	}

	i := p.msgInfo[id]
	if i.msgHandler != nil {
		i.msgHandler([]interface{}{msg, userData})
	}
	if i.msgRouter != nil {
		i.msgRouter.Go(msgType, msg, userData)
	}
	return nil
}

// goroutine safe
func (p *Processor) Unmarshal(data []byte) (proto.Message, error) {
	if len(data) < 2 {
		return nil, errors.New("protobuf data too short")
	}

	// id
	var id uint16
	if p.littleEndian {
		id = binary.LittleEndian.Uint16(data)
	} else {
		id = binary.BigEndian.Uint16(data)
	}

	// msg
	if id >= uint16(len(p.msgInfo)) {
		return nil, fmt.Errorf("message id %v not registered", id)
	}
	msg := reflect.New(p.msgInfo[id].msgType.Elem()).Interface().(proto.Message)
	return msg, proto.UnmarshalMerge(data[2:], msg)
}

// goroutine safe
func (p *Processor) Marshal(msg proto.Message) (id []byte, data []byte, err error) {
	msgType := reflect.TypeOf(msg)

	// id
	_id, ok := p.msgID[msgType]
	if !ok {
		err = fmt.Errorf("message %s not registered", msgType)
		return
	}

	id = make([]byte, 2)
	if p.littleEndian {
		binary.LittleEndian.PutUint16(id, _id)
	} else {
		binary.BigEndian.PutUint16(id, _id)
	}

	// data
	data, err = proto.Marshal(msg)
	return
}

// goroutine safe
func (p *Processor) Range(f func(id uint16, t reflect.Type)) {
	for id, i := range p.msgInfo {
		f(uint16(id), i.msgType)
	}
}
