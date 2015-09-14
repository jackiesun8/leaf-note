package conf

var (
	// gate conf 网关配置
	Encoding               = "json" // "json" or "protobuf"
	PendingWriteNum        = 2000
	LenMsgLen              = 2
	MinMsgLen       uint32 = 2
	MaxMsgLen       uint32 = 4096
	LittleEndian           = false

	// skeleton conf 骨架配置
	GoLen              = 10000
	TimerDispatcherLen = 10000
	ChanRPCLen         = 10000
)
