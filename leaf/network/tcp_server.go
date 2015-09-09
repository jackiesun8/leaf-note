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
	LenMsgLen    int        //
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
	msgParser := NewMsgParser()
	msgParser.SetMsgLen(server.LenMsgLen, server.MinMsgLen, server.MaxMsgLen)
	msgParser.SetByteOrder(server.LittleEndian)
	server.msgParser = msgParser
}

func (server *TCPServer) run() {
	for {
		conn, err := server.ln.Accept()
		if err != nil {
			if server.closeFlag {
				return
			} else {
				log.Error("accept error: %v", err)
				continue
			}
		}

		server.mutexConns.Lock()
		if len(server.conns) >= server.MaxConnNum {
			server.mutexConns.Unlock()
			conn.Close()
			log.Debug("too many connections")
			continue
		}
		server.conns[conn] = struct{}{}
		server.mutexConns.Unlock()

		server.wg.Add(1)

		tcpConn := newTCPConn(conn, server.PendingWriteNum, server.msgParser)
		agent := server.NewAgent(tcpConn)
		go func() {
			agent.Run()

			// cleanup
			tcpConn.Close()
			server.mutexConns.Lock()
			delete(server.conns, conn)
			server.mutexConns.Unlock()
			agent.OnClose()

			server.wg.Done()
		}()
	}
}

//关闭TCP服务器函数
func (server *TCPServer) Close() {
	server.closeFlag = true
	server.ln.Close()

	server.mutexConns.Lock()
	for conn, _ := range server.conns {
		conn.Close()
	}
	server.conns = make(ConnSet)
	server.mutexConns.Unlock()

	server.wg.Wait()
}
