package network

import (
	"github.com/name5566/leaf/log"
	"net"
	"sync"
)

//连接集合
type ConnSet map[net.Conn]struct{} //值为空结构体

//TCP连接类型定义
type TCPConn struct {
	sync.Mutex             //匿名字段
	conn       net.Conn    //底层连接
	writeChan  chan []byte //发送缓冲
	closeFlag  bool        //关闭标志
	msgParser  *MsgParser  //消息解析器
}

//新建TCP连接
func newTCPConn(conn net.Conn, pendingWriteNum int, msgParser *MsgParser) *TCPConn {
	tcpConn := new(TCPConn)                                //创建一个TCP连接实例
	tcpConn.conn = conn                                    //保存底层连接
	tcpConn.writeChan = make(chan []byte, pendingWriteNum) //创建发送缓冲区
	tcpConn.msgParser = msgParser                          //保存消息解析器

	go func() { //在一个新的goroutine中做发送数据工作
		for b := range tcpConn.writeChan { //如果发送缓冲区被关闭，此循环会自动结束（结束阻塞），如果没有数据，会阻塞在这里
			if b == nil { //如果收到的值为nil，而不是字节切片
				break //中断循环
			}

			_, err := conn.Write(b) //发送数据
			if err != nil {         //发生错误
				break //中断循环
			}
		}
		//清理工作
		conn.Close()             //关闭底层连接
		tcpConn.Lock()           //加锁
		tcpConn.closeFlag = true //设置关闭标志
		tcpConn.Unlock()         //解锁
	}()

	return tcpConn
}

func (tcpConn *TCPConn) doDestroy() {
	tcpConn.conn.(*net.TCPConn).SetLinger(0)
	tcpConn.conn.Close()
	close(tcpConn.writeChan)
	tcpConn.closeFlag = true
}

func (tcpConn *TCPConn) Destroy() {
	tcpConn.Lock()
	defer tcpConn.Unlock()
	if tcpConn.closeFlag {
		return
	}

	tcpConn.doDestroy()
}

func (tcpConn *TCPConn) Close() {
	tcpConn.Lock()         //上锁
	defer tcpConn.Unlock() //延迟解锁(保证Close执行结束前解锁)
	if tcpConn.closeFlag { //如果已经设置了关闭标志
		return //直接返回
	}

	tcpConn.doWrite(nil)
	tcpConn.closeFlag = true //设置关闭标志
}

func (tcpConn *TCPConn) doWrite(b []byte) {
	if len(tcpConn.writeChan) == cap(tcpConn.writeChan) {
		log.Debug("close conn: channel full")
		tcpConn.doDestroy()
		return
	}

	tcpConn.writeChan <- b
}

// b must not be modified by other goroutines
func (tcpConn *TCPConn) Write(b []byte) {
	tcpConn.Lock()
	defer tcpConn.Unlock()
	if tcpConn.closeFlag || b == nil {
		return
	}

	tcpConn.doWrite(b)
}

//实现io.Reader接口
func (tcpConn *TCPConn) Read(b []byte) (int, error) {
	return tcpConn.conn.Read(b)
}

func (tcpConn *TCPConn) LocalAddr() net.Addr {
	return tcpConn.conn.LocalAddr()
}

func (tcpConn *TCPConn) RemoteAddr() net.Addr {
	return tcpConn.conn.RemoteAddr()
}

func (tcpConn *TCPConn) ReadMsg() ([]byte, error) {
	return tcpConn.msgParser.Read(tcpConn)
}

func (tcpConn *TCPConn) WriteMsg(args ...[]byte) error {
	return tcpConn.msgParser.Write(tcpConn, args...)
}
