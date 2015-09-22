package gate

type Agent interface {
	WriteMsg(msg interface{})     //发送消息
	Close()                       //关闭代理
	UserData() interface{}        //获取用户数据
	SetUserData(data interface{}) //设置用户数据
}
