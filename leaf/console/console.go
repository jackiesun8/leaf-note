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
	reader *bufio.Reader //封装io.Reader or io.Writer对象，创建另外一个实现了对应接口的对象，提供缓存和文本读取的功能
}

//创建代理函数定义
//传入TCP连接封装
//返回满足network.Agent接口的对象
func newAgent(conn *network.TCPConn) network.Agent {
	a := new(Agent)                  //新建代理(定义在上面)
	a.conn = conn                    //保存TCP连接封装
	a.reader = bufio.NewReader(conn) //新建reader(带缓冲)
	return a
}

//实现代理接口(network.Agent)Run函数
//命令格式为 命令名 命令参数1 命令参数2 .... 命令参数n
func (a *Agent) Run() {
	for { //死循环
		if conf.ConsolePrompt != "" { //如果提示符不为空
			a.conn.Write([]byte(conf.ConsolePrompt)) //发送提示符
		}

		line, err := a.reader.ReadString('\n') //读取一个字符串，以\n分隔
		if err != nil {                        //读取出错
			break //退出循环
		}
		//在windows系统下，回车换行符号是"\r\n".但是在Linux等系统下是没有"\r"符号的
		line = strings.TrimSuffix(line[:len(line)-1], "\r") //line[:len(line)-1]去除\n,TrimSuffix去除\r

		args := strings.Fields(line) //按空格分割字符串为多个子字符串
		if len(args) == 0 {          //line只包含空格时args为空
			continue
		}
		if args[0] == "quit" { //如果第一个子字符串为quit
			break //退出循环
		}
		var c Command
		for _, _c := range commands { //遍历所有命令
			if _c.name() == args[0] { //匹配到某命令
				c = _c //取得命令引用
				break  //跳出匹配
			}
		}
		if c == nil { //未匹配到任何命令
			a.conn.Write([]byte("command not found, try `help` for help\r\n")) //发送命令未找到消息
			continue
		}
		output := c.run(args[1:]) //执行命令，参数为除了第一个子字符串（命令名）的剩余子字符串
		if output != "" {         //执行命令结果不为空
			a.conn.Write([]byte(output + "\r\n")) //发送命令执行结果
		}
	}
}

//实现代理接口OnClose函数
func (a *Agent) OnClose() {}
