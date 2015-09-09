package console

import (
	"bufio"
	"github.com/name5566/leaf/conf"
	"github.com/name5566/leaf/network"
	"math"
	"strconv"
	"strings"
)

var server *network.TCPServer

//初始化函数
func Init() {
	if conf.ConsolePort == 0 { //默认端口为0，不启用
		return
	}

	server = new(network.TCPServer)                             //创建一个tcp服务器
	server.Addr = "localhost:" + strconv.Itoa(conf.ConsolePort) //IP + 端口
	server.MaxConnNum = int(math.MaxInt32)                      //最大连接数
	server.PendingWriteNum = 100                                //发送缓冲区长度
	server.NewAgent = newAgent                                  //创建代理函数

	server.Start() //启动服务器
}

//销毁函数
func Destroy() {
	if server != nil {
		server.Close() //关闭TCP服务器
	}
}

//代理类型定义
type Agent struct {
	conn   *network.TCPConn
	reader *bufio.Reader
}

//创建代理函数定义
func newAgent(conn *network.TCPConn) network.Agent {
	a := new(Agent)                  //新建代理
	a.conn = conn                    //保存TCP连接
	a.reader = bufio.NewReader(conn) //新建reader(带缓冲)
	return a
}

//实现代理接口(network.Agent)Run函数
func (a *Agent) Run() {
	for { //死循环
		if conf.ConsolePrompt != "" {
			a.conn.Write([]byte(conf.ConsolePrompt))
		}

		line, err := a.reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSuffix(line[:len(line)-1], "\r")

		args := strings.Fields(line)
		if len(args) == 0 {
			continue
		}
		if args[0] == "quit" {
			break
		}
		var c Command
		for _, _c := range commands {
			if _c.name() == args[0] {
				c = _c
				break
			}
		}
		if c == nil {
			a.conn.Write([]byte("command not found, try `help` for help\r\n"))
			continue
		}
		output := c.run(args[1:])
		if output != "" {
			a.conn.Write([]byte(output + "\r\n"))
		}
	}
}

//实现代理接口OnClose函数
func (a *Agent) OnClose() {}
