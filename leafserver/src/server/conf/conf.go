package conf

var (
	// gate conf
	Encoding               = "json" // 编码方式定义，"json" or "protobuf"
	PendingWriteNum        = 2000
	LenMsgLen              = 2
	MinMsgLen       uint32 = 2
	MaxMsgLen       uint32 = 4096
	LittleEndian           = false

	// skeleton conf
	GoLen              = 10000
	TimerDispatcherLen = 10000
	ChanRPCLen         = 10000
)
