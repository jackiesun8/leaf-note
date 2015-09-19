package internal

import (
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"server/conf"
	"server/game"
	"server/msg"
)

//模块类型定义
type Module struct {
	*gate.TCPGate //匿名组合leaf框架的TCP网关
}

//模块初始化
func (m *Module) OnInit() {
	m.TCPGate = &gate.TCPGate{
		Addr:            conf.Server.Addr,
		MaxConnNum:      conf.Server.MaxConnNum,
		PendingWriteNum: conf.PendingWriteNum,
		LenMsgLen:       conf.LenMsgLen,
		MinMsgLen:       conf.MinMsgLen,
		MaxMsgLen:       conf.MaxMsgLen,
		LittleEndian:    conf.LittleEndian,
		AgentChanRPC:    game.ChanRPC,
	} //创建TCP网关

	//根据Encoding配置设置消息处理器
	switch conf.Encoding {
	case "json": //使用JSON处理消息
		m.TCPGate.JSONProcessor = msg.JSONProcessor
	case "protobuf": //使用protobuf处理消息
		m.TCPGate.ProtobufProcessor = msg.ProtobufProcessor
	default:
		log.Fatal("unknown encoding: %v", conf.Encoding) //未知设置
	}
}
