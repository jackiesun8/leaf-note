package conf

var (
	LenStackBuf = 4096 //保存stack trace buf长度

	LogLevel string //日志级别
	LogPath  string //日志路径

	ConsolePort   int    //控制台端口，默认不开启
	ConsolePrompt string = "Leaf# "
	ProfilePath   string
)
