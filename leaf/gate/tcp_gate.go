package gate

import (
	"github.com/golang/protobuf/proto"
	"github.com/name5566/leaf/chanrpc"
	"github.com/name5566/leaf/log"
	"github.com/name5566/leaf/network"
	"github.com/name5566/leaf/network/json"
	"github.com/name5566/leaf/network/protobuf"
	"reflect"
)

//TCP网关服务器类型定义
type TCPGate struct {
	Addr              string              //地址
	MaxConnNum        int                 //最大连接数
	PendingWriteNum   int                 //发送缓冲区长度
	LenMsgLen         int                 //消息长度占用字节数
	MinMsgLen         uint32              //最小消息长度
	MaxMsgLen         uint32              //最大消息长度
	LittleEndian      bool                //大小端标志
	JSONProcessor     *json.Processor     //json处理器
	ProtobufProcessor *protobuf.Processor //protobuf处理器
	AgentChanRPC      *chanrpc.Server     //RPC服务器
}

//实现了Module接口的Run
func (gate *TCPGate) Run(closeSig chan bool) {
	server := new(network.TCPServer) //创建TCP服务器
	//设置TCP服务器相关参数
	server.Addr = gate.Addr
	server.MaxConnNum = gate.MaxConnNum
	server.PendingWriteNum = gate.PendingWriteNum
	server.LenMsgLen = gate.LenMsgLen
	server.MinMsgLen = gate.MinMsgLen
	server.MaxMsgLen = gate.MaxMsgLen
	server.LittleEndian = gate.LittleEndian
	server.NewAgent = func(conn *network.TCPConn) network.Agent { //设置创建代理函数
		a := new(TCPAgent) //创建TCP代理
		a.conn = conn      //保存TCP连接
		a.gate = gate      //保存TCP网关

		if gate.AgentChanRPC != nil {
			gate.AgentChanRPC.Go("NewAgent", a)
		}

		return a
	}

	server.Start() //启动TCP服务器
	<-closeSig     //等待关闭信号
	server.Close() //关闭TCP服务器
}

//Module接口的OnDestroy
func (gate *TCPGate) OnDestroy() {}

//TCP代理类型定义
type TCPAgent struct {
	conn     *network.TCPConn //TCP连接
	gate     *TCPGate         //TCP网关
	userData interface{}      //用户数据
}

//实现代理接口(network.Agent)Run函数
func (a *TCPAgent) Run() {
	for {
		data, err := a.conn.ReadMsg() //读取一条完整的消息
		if err != nil {
			log.Debug("read message error: %v", err)
			break
		}

		if a.gate.JSONProcessor != nil { //配置为使用JSON处理
			// json
			msg, err := a.gate.JSONProcessor.Unmarshal(data) //解码JSON数据
			if err != nil {
				log.Debug("unmarshal json error: %v", err)
				break
			}
			err = a.gate.JSONProcessor.Route(msg, Agent(a)) //分发数据，将a转化成Agent作为用户数据
			if err != nil {
				log.Debug("route message error: %v", err)
				break
			}
		} else if a.gate.ProtobufProcessor != nil { //配置为使用protobuf处理
			// protobuf
			msg, err := a.gate.ProtobufProcessor.Unmarshal(data) //解码protobuf数据
			if err != nil {
				log.Debug("unmarshal protobuf error: %v", err)
				break
			}
			err = a.gate.ProtobufProcessor.Route(msg, Agent(a)) //分发数据
			if err != nil {
				log.Debug("route message error: %v", err)
				break
			}
		}
	}
}

//实现代理接口(network.Agent)OnClose函数
func (a *TCPAgent) OnClose() {
	if a.gate.AgentChanRPC != nil {
		err := a.gate.AgentChanRPC.Open(0).Call0("CloseAgent", a)
		if err != nil {
			log.Error("chanrpc error: %v", err)
		}
	}
}

//实现代理接口(gate.Agent)WriteMsg函数
//发送消息
func (a *TCPAgent) WriteMsg(msg interface{}) {
	if a.gate.JSONProcessor != nil { //使用JSON处理器
		// json
		data, err := a.gate.JSONProcessor.Marshal(msg) //编码JSON消息
		if err != nil {
			log.Error("marshal json %v error: %v", reflect.TypeOf(msg), err)
			return
		}
		a.conn.WriteMsg(data) //发送消息
	} else if a.gate.ProtobufProcessor != nil { //使用protobuf处理器
		// protobuf
		id, data, err := a.gate.ProtobufProcessor.Marshal(msg.(proto.Message)) //编码protobuf消息
		if err != nil {
			log.Error("marshal protobuf %v error: %v", reflect.TypeOf(msg), err)
			return
		}
		a.conn.WriteMsg(id, data) //发送消息
	}
}

//实现代理接口(gate.Agent)Close函数
//关闭代理
func (a *TCPAgent) Close() {
	a.conn.Close() //关闭连接
}

//实现代理接口(gate.Agent)UserData函数
//获取用户数据
func (a *TCPAgent) UserData() interface{} {
	return a.userData
}

//实现代理接口(gate.Agent)SetUserData函数
//设置用户数据
func (a *TCPAgent) SetUserData(data interface{}) {
	a.userData = data
}
