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
	if len(tcpConn.writeChan) == cap(tcpConn.writeChan) { //如果发送缓冲区的长度等于最大容量
		log.Debug("close conn: channel full") //日志记录，管道已满
		tcpConn.doDestroy()                   //做销毁操作
		return
	}

	tcpConn.writeChan <- b //将待发数据发送到发送缓冲区
}

// b must not be modified by other goroutines
func (tcpConn *TCPConn) Write(b []byte) {
	tcpConn.Lock()                     //加锁
	defer tcpConn.Unlock()             //延迟解锁
	if tcpConn.closeFlag || b == nil { //如果连接已关闭或者传入的b为空
		return //返回
	}

	tcpConn.doWrite(b) //做具体的发送操作
}

//实现io.Reader接口
//将被bufio封装
func (tcpConn *TCPConn) Read(b []byte) (int, error) {
	return tcpConn.conn.Read(b) //调用底层conn读取数据
}

//返回本地地址
func (tcpConn *TCPConn) LocalAddr() net.Addr {
	return tcpConn.conn.LocalAddr()
}

//返回远程(客户端)地址
func (tcpConn *TCPConn) RemoteAddr() net.Addr {
	return tcpConn.conn.RemoteAddr()
}

//读取消息
func (tcpConn *TCPConn) ReadMsg() ([]byte, error) {
	return tcpConn.msgParser.Read(tcpConn) //使用消息解析器读取
}

//发送消息
func (tcpConn *TCPConn) WriteMsg(args ...[]byte) error {
	return tcpConn.msgParser.Write(tcpConn, args...) //使用消息解析器发送
}
