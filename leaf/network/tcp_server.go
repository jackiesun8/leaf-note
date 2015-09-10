package network

import (
	"github.com/name5566/leaf/log"
	"net"
	"sync"
)

//TCP服务器类型定义
type TCPServer struct {
	Addr            string               //地址
	MaxConnNum      int                  //最大连接数
	PendingWriteNum int                  //发送缓冲区长度
	NewAgent        func(*TCPConn) Agent //创建代理函数
	ln              net.Listener         //监听连接器
	conns           ConnSet              //连接集合
	mutexConns      sync.Mutex           //互斥锁
	wg              sync.WaitGroup       //等待组
	closeFlag       bool                 //关闭标志

	// msg parser 消息解析器
	LenMsgLen    int        //消息长度的长度(len)
	MinMsgLen    uint32     //最小消息长度
	MaxMsgLen    uint32     //最大消息长度
	LittleEndian bool       //大小端标志
	msgParser    *MsgParser //消息解析器
}

//启动TCP服务器
func (server *TCPServer) Start() {
	server.init()   //初始化
	go server.run() //在一个goroutine里运行TCP服务器
}

//初始化TCP服务器
func (server *TCPServer) init() {
	ln, err := net.Listen("tcp", server.Addr) //监听
	if err != nil {
		log.Fatal("%v", err)
	}

	if server.MaxConnNum <= 0 { //最大连接数小于0，重置到100
		server.MaxConnNum = 100
		log.Release("invalid MaxConnNum, reset to %v", server.MaxConnNum)
	}
	if server.PendingWriteNum <= 0 { //发送缓冲区长度小于0，重置到100
		server.PendingWriteNum = 100
		log.Release("invalid PendingWriteNum, reset to %v", server.PendingWriteNum)
	}
	if server.NewAgent == nil { //创建代理函数不能为空
		log.Fatal("NewAgent must not be nil")
	}

	server.ln = ln               //保存监听连接器
	server.conns = make(ConnSet) //创建连接集合
	server.closeFlag = false     //关闭标志

	// msg parser
	msgParser := NewMsgParser()                                               //创建消息解析器
	msgParser.SetMsgLen(server.LenMsgLen, server.MinMsgLen, server.MaxMsgLen) //设置消息长度
	msgParser.SetByteOrder(server.LittleEndian)                               //设置字节序
	server.msgParser = msgParser                                              //保存消息解析器
}

//运行TCP服务器
func (server *TCPServer) run() {
	for { //死循环
		conn, err := server.ln.Accept() //接受一个连接
		if err != nil {                 //如果有错(如果调用了server.Close,再Accept,那么就会出错)
			if server.closeFlag { //服务器关闭标志为true 因为关闭了TCP服务器导致的出错
				return //结束循环
			} else {
				log.Error("accept error: %v", err) //日志记录接受连接出错
				continue                           //继续循环
			}
		}

		server.mutexConns.Lock()                    //加锁，为什么要加锁，因为会从不同的goroutine中访问server.conns,比如从外部goroutine中调用server.Close或者在新的goroutine中运行代理执行清理工作的时候或者当前for循环所在goroutine中增加连接记录
		if len(server.conns) >= server.MaxConnNum { //如果当前连接数超过上限
			server.mutexConns.Unlock()        //解锁
			conn.Close()                      //关闭新来的连接
			log.Debug("too many connections") //日志记录：太多连接了
			continue                          //继续循环
		}
		//增加连接记录
		server.conns[conn] = struct{}{} //struct{}为类型，第二个{}为初始化，只不过是空值而已
		server.mutexConns.Unlock()      //解锁

		server.wg.Add(1) //等待组+1

		tcpConn := newTCPConn(conn, server.PendingWriteNum, server.msgParser) //创建一个TCP连接(原有net.Conn的封装)
		agent := server.NewAgent(tcpConn)                                     //调用注册的创建代理函数创建代理
		go func() {                                                           //此处形成闭包
			agent.Run() //在一个新的goroutine中运行代理
			//执行到这里时agent.Run for循环结束
			// cleanup
			//清理工作
			tcpConn.Close()            //关闭连接（封装层）
			server.mutexConns.Lock()   //加锁
			delete(server.conns, conn) //从连接集合中删除连接
			server.mutexConns.Unlock() //解锁
			agent.OnClose()            //关闭代理

			server.wg.Done() //等待组-1
		}()
	}
}

//关闭TCP服务器函数
//疑问？如果关闭了TCP服务器，那创建的那些与客户端的连接是如何关闭的
func (server *TCPServer) Close() {
	server.closeFlag = true //设置关闭标记
	server.ln.Close()       //关闭监听器,导致再Accept时出错

	server.mutexConns.Lock()            //加锁
	for conn, _ := range server.conns { //遍历现有连接
		conn.Close() //关闭连接(底层)
	}
	server.conns = make(ConnSet) //重置连接集合
	server.mutexConns.Unlock()   //解锁

	server.wg.Wait() //等待所有goroutine退出
}
