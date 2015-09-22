package mongodb

import (
	"container/heap"
	"github.com/name5566/leaf/log"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"sync"
)

// session
//会话类型定义
type Session struct {
	*mgo.Session     //mgo的会话
	ref          int //引用
	index        int //会话索引
}

// session heap
//会话堆类型定义（连接池）
type SessionHeap []*Session //会话指针切片

//实现sort.Interface的Len方法
//Len is the number of elements in the collection
func (h SessionHeap) Len() int {
	return len(h)
}

//实现sort.Interface的Less方法方法
// Less reports whether the element with
// index i should sort before the element with index j.
func (h SessionHeap) Less(i, j int) bool {
	return h[i].ref < h[j].ref
}

//实现sort.Interface的Swap方法
//Swap swaps the elements with indexes i and j.
func (h SessionHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

//实现了heap的Push方法
func (h *SessionHeap) Push(s interface{}) {
	s.(*Session).index = len(*h)
	*h = append(*h, s.(*Session))
}

//实现了heap的Pop方法
func (h *SessionHeap) Pop() interface{} {
	l := len(*h)
	s := (*h)[l-1]
	s.index = -1
	*h = (*h)[:l-1]
	return s
}

//拨号上下文
type DialContext struct {
	sync.Mutex             //互斥锁
	sessions   SessionHeap //会话堆
}

// goroutine safe
//连接mongodb数据库，返回拨号上下文
func Dial(url string, sessionNum int) (*DialContext, error) {
	if sessionNum <= 0 { //非法会话数
		sessionNum = 100 //重置为100
		log.Release("invalid sessionNum, reset to %v", sessionNum)
	}

	s, err := mgo.Dial(url) //连接数据库
	if err != nil {
		return nil, err
	}

	c := new(DialContext) //创建拨号上下文

	// sessions
	c.sessions = make(SessionHeap, sessionNum) //创建会话堆
	c.sessions[0] = &Session{s, 0, 0}          //保存会话，0索引
	for i := 1; i < sessionNum; i++ {
		c.sessions[i] = &Session{s.New(), 0, i} //利用原始会话创建新的会话。1，2，3，i索引
	}
	heap.Init(&c.sessions) //A heap must be initialized before any of the heap operations can be used

	return c, nil
}

// goroutine safe
//关闭拨号上下文
func (c *DialContext) Close() {
	c.Lock()
	for _, s := range c.sessions {
		s.Close()
		if s.ref != 0 {
			log.Error("session ref = %v", s.ref)
		}
	}
	c.Unlock()
}

// goroutine safe
//引用，以获取一个会话
func (c *DialContext) Ref() *Session {
	c.Lock()           //上下文加锁
	s := c.sessions[0] //取出一个会话
	if s.ref == 0 {    //会话引用为0
		s.Refresh() //刷新会话
	}
	s.ref++                  //增加会话引用
	heap.Fix(&c.sessions, 0) //重建堆序，比删除再添加一个新元素更廉价
	c.Unlock()               //上下文解锁

	return s //返回一个会话
}

// goroutine safe
//取消会话引用
func (c *DialContext) UnRef(s *Session) {
	c.Lock()                       //加锁
	s.ref--                        //减少引用
	heap.Fix(&c.sessions, s.index) //重建堆序
	c.Unlock()                     //解锁
}

// goroutine safe
// 创建自增字段
func (c *DialContext) EnsureCounter(db string, collection string, id string) error {
	s := c.Ref()     //取得一个会话
	defer c.UnRef(s) //延迟释放会话

	err := s.DB(db).C(collection).Insert(bson.M{
		"_id": id,
		"seq": 0,
	})
	if mgo.IsDup(err) { //判断错误是否是duplicate key error
		return nil
	} else {
		return err
	}
}

// goroutine safe
//返回自增字段的下一个值seq
func (c *DialContext) NextSeq(db string, collection string, id string) (int, error) {
	s := c.Ref()     //取得一个会话
	defer c.UnRef(s) //延迟释放会话

	var res struct { //定义一个结构体变量保存返回值
		Seq int
	}
	_, err := s.DB(db).C(collection).FindId(id).Apply(mgo.Change{ //查找id的document并执行更新操作
		Update:    bson.M{"$inc": bson.M{"seq": 1}}, //自增seq的值
		ReturnNew: true,                             //返回新值
	}, &res)

	return res.Seq, err
}

// goroutine safe
//创建索引
func (c *DialContext) EnsureIndex(db string, collection string, key []string) error {
	s := c.Ref()
	defer c.UnRef(s)

	return s.DB(db).C(collection).EnsureIndex(mgo.Index{
		Key:    key,   //包含了所有字段
		Unique: false, //非唯一
		Sparse: true,  //只有包含了key中的字段的document才会包含进index
	})
}

// goroutine safe
//创建唯一索引
func (c *DialContext) EnsureUniqueIndex(db string, collection string, key []string) error {
	s := c.Ref()
	defer c.UnRef(s)

	return s.DB(db).C(collection).EnsureIndex(mgo.Index{
		Key:    key,  //包含了所有字段
		Unique: true, //唯一索引
		Sparse: true, //只有包含了key中的字段的document才会包含进index
	})
}
