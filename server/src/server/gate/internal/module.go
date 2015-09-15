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
	*gate.TCPGate //匿名组合TCP网关
}

//初始化
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

	switch conf.Encoding {
	case "json":
		m.TCPGate.JSONProcessor = msg.JSONProcessor
	case "protobuf":
		m.TCPGate.ProtobufProcessor = msg.ProtobufProcessor
	default:
		log.Fatal("unknown encoding: %v", conf.Encoding)
	}
}
