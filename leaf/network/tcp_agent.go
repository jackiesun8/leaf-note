package network

//代理接口
type Agent interface {
	Run()     //运行函数
	OnClose() //关闭函数
}
