package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/name5566/leaf/chanrpc"
	"github.com/name5566/leaf/log"
	"reflect"
)

//处理器类型定义
type Processor struct {
	msgInfo map[string]*MsgInfo //消息信息映射
}

//消息信息类型定义
type MsgInfo struct {
	msgType    reflect.Type    //消息类型
	msgRouter  *chanrpc.Server //处理消息的RPC服务器
	msgHandler MsgHandler      //消息处理函数，处理消息有两种方式，一种的RPC服务器，一种是处理函数，可以同时处理
}

type MsgHandler func([]interface{})

//创建一个处理器
func NewProcessor() *Processor {
	p := new(Processor)                   //创建处理器
	p.msgInfo = make(map[string]*MsgInfo) //创建消息信息映射
	return p
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
// 注册消息
func (p *Processor) Register(msg interface{}) {
	msgType := reflect.TypeOf(msg)                       //获取消息的类型
	if msgType == nil || msgType.Kind() != reflect.Ptr { //判断消息的合法性，不能为空，需要是指针
		log.Fatal("json message pointer required")
	}
	msgID := msgType.Elem().Name() //获取消息类型本身（不是指针）的名字，作为消息ID
	if msgID == "" {
		log.Fatal("unnamed json message")
	}
	if _, ok := p.msgInfo[msgID]; ok { //判断消息是否已经注册
		log.Fatal("message %v is already registered", msgID)
	}

	i := new(MsgInfo)    //新建一个消息信息
	i.msgType = msgType  //保存消息类型
	p.msgInfo[msgID] = i //保存消息信息到映射中
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
//设置路由
func (p *Processor) SetRouter(msg interface{}, msgRouter *chanrpc.Server) {
	msgType := reflect.TypeOf(msg)                       //获取消息类型
	if msgType == nil || msgType.Kind() != reflect.Ptr { //判断消息合法性
		log.Fatal("json message pointer required")
	}
	msgID := msgType.Elem().Name() //获取消息类型本身的名字，也就是消息ID
	i, ok := p.msgInfo[msgID]      //根据消息ID获得消息信息
	if !ok {                       //获取消息信息失败
		log.Fatal("message %v not registered", msgID)
	}

	i.msgRouter = msgRouter //保存RPC服务器引用
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
//设置消息处理函数
func (p *Processor) SetHandler(msg interface{}, msgHandler MsgHandler) {
	msgType := reflect.TypeOf(msg)                       //消息类型
	if msgType == nil || msgType.Kind() != reflect.Ptr { //判断合法性
		log.Fatal("json message pointer required")
	}
	msgID := msgType.Elem().Name() //获取消息ID
	i, ok := p.msgInfo[msgID]      //获取消息信息
	if !ok {
		log.Fatal("message %v not registered", msgID)
	}

	i.msgHandler = msgHandler //保存消息处理函数
}

// goroutine safe
//路由
func (p *Processor) Route(msg interface{}, userData interface{}) error {
	msgType := reflect.TypeOf(msg)                       //获取消息类型
	if msgType == nil || msgType.Kind() != reflect.Ptr { //判断合法性
		return errors.New("json message pointer required")
	}
	msgID := msgType.Elem().Name() //获取消息ID
	i, ok := p.msgInfo[msgID]      //获取消息信息
	if !ok {                       //获取失败
		return fmt.Errorf("message %v not registered", msgID)
	}

	if i.msgHandler != nil { //调用消息处理函数
		i.msgHandler([]interface{}{msg, userData})
	}
	if i.msgRouter != nil { //调用RPC服务器
		i.msgRouter.Go(msgType, msg, userData)
	}
	return nil
}

// goroutine safe
//解码消息
func (p *Processor) Unmarshal(data []byte) (interface{}, error) {
	var m map[string]json.RawMessage
	err := json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	if len(m) != 1 {
		return nil, errors.New("invalid json data")
	}

	for msgID, data := range m {
		i, ok := p.msgInfo[msgID]
		if !ok {
			return nil, fmt.Errorf("message %v not registered", msgID)
		}

		// msg
		msg := reflect.New(i.msgType.Elem()).Interface()
		return msg, json.Unmarshal(data, msg)
	}

	panic("bug")
}

// goroutine safe
//编码消息
func (p *Processor) Marshal(msg interface{}) ([]byte, error) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		return nil, errors.New("json message pointer required")
	}
	msgID := msgType.Elem().Name()
	if _, ok := p.msgInfo[msgID]; !ok {
		return nil, fmt.Errorf("message %v not registered", msgID)
	}

	// data
	m := map[string]interface{}{msgID: msg}
	return json.Marshal(m)
}
