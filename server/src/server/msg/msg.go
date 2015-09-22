package msg

import (
	"github.com/name5566/leaf/network/json"
	"github.com/name5566/leaf/network/protobuf"
)

var (
	JSONProcessor     = json.NewProcessor()     //创建JSON处理器
	ProtobufProcessor = protobuf.NewProcessor() //创建protobuf处理器
)

//初始化
func init() {
	//注册消息
	JSONProcessor.Register(&S2C_Close{})
	JSONProcessor.Register(&C2S_Auth{})
	JSONProcessor.Register(&S2C_Auth{})
}

// Close
const (
	S2C_Close_LoginRepeated = 1
	S2C_Close_InnerError    = 2
)

type S2C_Close struct {
	Err int
}

// Auth
type C2S_Auth struct {
	AccID string
}

const (
	S2C_Auth_OK           = 0
	S2C_Auth_AccIDInvalid = 1
)

type S2C_Auth struct {
	Err int
}
