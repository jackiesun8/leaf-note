package network

//代理接口
type Agent interface {
	Run()
	OnClose()
}
