package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// 定义了客户端命令的解析类型
// type byte = uint8
type CmdType = byte

const (
	COMMAND_UNKNOWN CmdType = 0x00 // 未知类型
	COMMAND_INLINE  CmdType = 0x01 // 简单文本格式
	COMMAND_BULK    CmdType = 0x02 // 多条指令、结构化格式
)

const (
	// 都是客户端发送来的一次完整命令的最大长度限制
	GODIS_IO_BUF     = 1024 * 16 // I/O 缓冲区大小（16KB）
	GODIS_MAX_BULK   = 1024 * 4  // 最大 Bulk（结构化）命令个数 最多支持 4K 个参数的结构化命令
	GODIS_MAX_INLINE = 1024 * 4  // 最大 Inline 命令长度 单条 inline 文本命令最大长度为 4KB
)

type GodisDB struct {
	data   *Dict
	expire *Dict
}

type GodisServer struct {
	fd      int
	port    int
	db      *GodisDB
	clients map[int]*GodisClient // 维护的客户端列表
	aeLoop  *AeLoop
}

type GodisClient struct {
	fd       int
	db       *GodisDB // 指向 GodisServer 中的数据库实例
	args     []*Gobj  // 当前解析出的命令参数（比如 SET key value 拆成三项）
	reply    *List    // 回复缓冲区，等待发送给客户端的数据列表
	queryBuf []byte   // 读缓冲区，接收客户端发来的数据 ('h' -> 0x68 (104))
	sentLen  int      //
	queryLen int      // 当前缓冲区有效数据长度
	cmdTy    CmdType  // 当前客户端请求的命令类型（inline / bulk）
	bulkNum  int      // bulk 模式下预期参数数量
	bulkLen  int      // bulk 模式下当前读取的参数长度
}

// 定义命令和处理函数的映射关系
type CommandProc func(c *GodisClient)
type GodisCommand struct {
	name  string
	proc  CommandProc
	arity int
}

var server GodisServer
var cmdTable []GodisCommand = []GodisCommand{
	{"get", getCommand, 2},
	{"set", setCommand, 3},
	{"expire", expireCommand, 3},
	//TODO
}

func expireIfNeeded(key *Gobj) {
	entry := server.db.expire.Find(key)
	if entry == nil {
		return
	}
	when := entry.Val.IntVal()
	if when > GetMsTime() {
		return
	}
	server.db.expire.Delete(key)
	server.db.data.Delete(key)
}

func findKeyRead(key *Gobj) *Gobj {
	expireIfNeeded(key)
	return server.db.data.Get(key)
}

func getCommand(c *GodisClient) {
	key := c.args[1]
	val := findKeyRead(key)
	if val == nil {
		//TODO: extract shared.strings
		c.AddReplyStr("$-1\r\n")
	} else if val.Type_ != GSTR {
		//TODO: extract shared.strings
		c.AddReplyStr("-ERR: wrong type\r\n")
	} else {
		str := val.StrVal()
		c.AddReplyStr(fmt.Sprintf("$%d%v\r\n", len(str), str))
	}
}

func setCommand(c *GodisClient) {
	key := c.args[1]
	val := c.args[2]
	if val.Type_ != GSTR {
		//TODO: extract shared.strings
		c.AddReplyStr("-ERR: wrong type\r\n")
	}
	server.db.data.Set(key, val)
	server.db.expire.Delete(key)
	c.AddReplyStr("+OK\r\n")
}

func expireCommand(c *GodisClient) {
	key := c.args[1]
	val := c.args[2]
	if val.Type_ != GSTR {
		//TODO: extract shared.strings
		c.AddReplyStr("-ERR: wrong type\r\n")
	}
	expire := GetMsTime() + (val.IntVal() * 1000)
	expObj := CreateFromInt(expire)
	server.db.expire.Set(key, expObj)
	expObj.DecrRefCount()
	c.AddReplyStr("+OK\r\n")
}

func lookupCommand(cmdStr string) *GodisCommand {
	for _, c := range cmdTable {
		if c.name == cmdStr {
			return &c
		}
	}
	return nil
}

func (c *GodisClient) AddReply(o *Gobj) {
	c.reply.Append(o)
	o.IncrRefCount()
	server.aeLoop.AddFileEvent(c.fd, AE_WRITABLE, SendReplyToClient, c)
}

func (c *GodisClient) AddReplyStr(str string) {
	o := CreateObject(GSTR, str)
	c.AddReply(o)
	o.DecrRefCount() // 初始化后有一次ref 这里释放一次
}

func ProcessCommand(c *GodisClient) {
	cmdStr := c.args[0].StrVal()
	log.Printf("process command: %v\n", cmdStr)
	if cmdStr == "quit" {
		freeClient(c)
		return
	}
	cmd := lookupCommand(cmdStr)
	if cmd == nil {
		c.AddReplyStr("-ERR: unknow command")
		resetClient(c)
		return
	} else if cmd.arity != len(c.args) {
		c.AddReplyStr("-ERR: wrong number of args")
		resetClient(c)
		return
	}
	cmd.proc(c)
	resetClient(c)
}

// 具体的 CommandProc 实现
func freeArgs(client *GodisClient) {
	for _, v := range client.args {
		v.DecrRefCount()
	}
}

func freeReplyList(client *GodisClient) {
	for client.reply.length != 0 {
		n := client.reply.head
		client.reply.DelNode(n)
		n.Val.DecrRefCount()
	}
}

func freeClient(client *GodisClient) {
	freeArgs(client)
	delete(server.clients, client.fd)
	server.aeLoop.RemoveFileEvent(client.fd, AE_READABLE)
	server.aeLoop.RemoveFileEvent(client.fd, AE_WRITABLE)
	freeReplyList(client)
	unix.Close(client.fd)
}

func resetClient(client *GodisClient) {
	freeArgs(client)
	client.cmdTy = COMMAND_UNKNOWN
	client.bulkLen = 0
	client.bulkNum = 0
}

func (client *GodisClient) findLineInQuery() (int, error) {
	// 查找 \r\n
	for i := 0; i < client.queryLen-1; i++ {
		if client.queryBuf[i] == '\r' && client.queryBuf[i+1] == '\n' {
			return i, nil
		}
	}
	return -1, errors.New("find line error")
}

/*
找到最近的 \r\n	并处理这之前的数据：
流式传输 没有收全会直接 break
直到下一次回调触发
*/
// SET mykey hello\r\n
func handleInlineBuf(client *GodisClient) (bool, error) {
	// 每次只处理一条完整的指令
	index, err := client.findLineInQuery()
	// 如果命令不完整的则不会出现 /r\n
	if index < 0 {
		return false, err
	}
	// client.queryBuf 将 []byte->string
	subs := strings.Split(string(client.queryBuf[:index]), " ")
	client.queryBuf = client.queryBuf[index+2:]
	client.queryLen -= index + 2
	client.args = make([]*Gobj, len(subs))
	for i, v := range subs {
		client.args[i] = CreateObject(GSTR, v)
	}

	return true, nil
}

func (client *GodisClient) getNumInQuery(s, e int) (int, error) {
	num, err := strconv.Atoi(string(client.queryBuf[s:e])) // 转换为字符串并解析为十进制整数
	client.queryBuf = client.queryBuf[e+2:]
	client.queryLen -= e + 2
	return num, err
}

// *3\r\n$3\r\nSET\r\n$5\r\nmykey\r\n$5\r\nhello\r\n
/*
无论是
findLineInQuery / getNumInQuery
都需要找到 \r\n，同时完成 querybuf 的偏移
*/
func handleBulkBuf(client *GodisClient) (bool, error) {
	// bulkNum 代表需要从 querybuf 中读取的参数个数
	if client.bulkNum == 0 {
		index, err := client.findLineInQuery()
		if index < 0 {
			return false, err
		}

		bnum, err := client.getNumInQuery(1, index)
		if err != nil {
			return false, err
		}
		// 代表空字符串 SET key ""
		if bnum == 0 {
			return true, nil
		}
		client.bulkNum = bnum
		client.args = make([]*Gobj, bnum)
	}
	for client.bulkNum > 0 {
		// 从 querybuf 读多长
		if client.bulkLen == 0 {
			index, err := client.findLineInQuery()
			if index < 0 {
				return false, err
			}
			if client.queryBuf[0] != '$' {
				return false, errors.New("expect $ for bulk")
			}
			// 头和尾都被处理过，1, index
			blen, err := client.getNumInQuery(1, index)
			// 长度为 0 代表空字符串
			if err != nil || blen == 0 {
				return false, err
			}
			// GODIS_MAX_BULK -> 限制的是 $ 后、\r 前的 bulk string
			if blen > GODIS_MAX_BULK {
				return false, errors.New("bulk length too long")
			}
			client.bulkLen = blen
		}
		// 开始读取数据并加入到 args 中
		// 读不完，下次回调再读
		if client.queryLen < client.bulkLen+2 {
			return false, nil
		}
		index := client.bulkLen
		if client.queryBuf[index] != '\r' || client.queryBuf[index+1] != '\n' {
			return false, errors.New("expect CRLF for bulk end")
		}
		client.args[len(client.args)-client.bulkNum] = CreateObject(GSTR, string(client.queryBuf[:index]))
		client.queryBuf = client.queryBuf[index+2:]
		client.queryLen -= index + 2
		client.bulkLen = 0
		client.bulkNum -= 1
	}
	return true, nil
}

// 传递指针可以设置成员变量
func ProcessQueryBuf(client *GodisClient) error {
	for client.queryLen > 0 {
		// 初始时不知道cmd类型
		if client.cmdTy == COMMAND_UNKNOWN {
			if client.queryBuf[0] == '*' {
				client.cmdTy = COMMAND_BULK
			} else {
				client.cmdTy = COMMAND_INLINE
			}
		}
		var ok bool
		var err error

		// 将 querybuf 中的内容解析到 args 中
		if client.cmdTy == COMMAND_INLINE {
			ok, err = handleInlineBuf(client)
		} else if client.cmdTy == COMMAND_BULK {
			ok, err = handleBulkBuf(client)
		} else {
			return errors.New("unknown command type")
		}
		// 处理出错返回 error 被 ReadQueryFromClient 捕获后释放 *GodisClient
		if err != nil {
			return err
		}
		// 命令是否完整
		if ok {
			if len(client.args) == 0 {
				resetClient(client)
			} else {
				ProcessCommand(client)
			}
		} else {
			break
		}
	}
	return nil
}

// 处理客户端的命令
func ReadQueryFromClient(loop *AeLoop, fd int, extra interface{}) {
	client := extra.(*GodisClient) // 接口的 assert
	if len(client.queryBuf)-client.queryLen < GODIS_MAX_BULK {
		// func append(slice []T, elems ...T) []T 表示展开
		client.queryBuf = append(client.queryBuf, make([]byte, GODIS_MAX_BULK)...)
	}
	// 偏移 querylen 之后开始读数据
	n, err := unix.Read(fd, client.queryBuf[client.queryLen:])
	if err != nil {
		log.Println("read error:", err)
		freeClient(client)
		return
	}

	client.queryLen += n
	log.Printf("read %v bytes from client:%v\n", n, client.fd)
	log.Printf("ReadQueryFromClient, queryBuf : %v\n", string(client.queryBuf))

	// 处理数据
	err = ProcessQueryBuf(client)
	if err != nil {
		log.Printf("process query buf err: %v\n", err)
		freeClient(client)
		return
	}
}

func SendReplyToClient(loop *AeLoop, fd int, extra interface{}) {
	client := extra.(*GodisClient)
	log.Printf("SendReplyToClient, reply len:%v\n", client.reply.Length())
	for client.reply.Length() > 0 {
		rep := client.reply.First()
		buf := []byte(rep.Val.StrVal())
		bufLen := len(buf)
		// sentlen < buflen 上一次没有发完
		if client.sentLen < bufLen {
			n, err := unix.Write(fd, buf[client.sentLen:])
			if err != nil {
				log.Printf("send reply err: %v\n", err)
				freeClient(client)
				return
			}
			client.sentLen += n
			log.Printf("send %v bytes to client:%v\n", n, client.fd)
			/*
			 此时 head 已经发完了
			 client.reply 中的每条消息都一次性成功写完 则不break
			*/
			if client.sentLen == bufLen {
				client.reply.DelNode(rep)
				rep.Val.DecrRefCount()
				client.sentLen = 0
			} else {
				// 说明此时还是没有发完 缓冲区当中没有 bytes 了
				break
			}
		}
	}
	//
	if client.reply.Length() == 0 {
		client.sentLen = 0
		loop.RemoveFileEvent(fd, AE_WRITABLE)
	}
}
