package conf

import (
	"encoding/json"
	"github.com/name5566/leaf/log"
	"io/ioutil"
)

var Server struct {
	LogLevel   string
	LogPath    string
	Addr       string
	MaxConnNum int
} //定义一个Server结构变量用来存储服务器一些配置

func init() {
	data, err := ioutil.ReadFile("conf/server.json") //读取服务器配置 位于leafserver/bin/conf/server.json
	if err != nil {
		log.Fatal("%v", err)
	}
	err = json.Unmarshal(data, &Server) //解析数据到Server中
	if err != nil {
		log.Fatal("%v", err)
	}
}
