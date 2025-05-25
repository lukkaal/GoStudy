# Go Modules 项目结构与依赖管理总结

## 基本概念回顾与拓展

- **go mod（模块化管理）**  
  每个含 `go.mod` 的目录即为一个模块（Module）。  
  不再依赖 GOPATH，模块可位于任意目录。  
  用于声明模块名及所有依赖和版本（require），可通过 replace 重定向本地模块路径。

- **package（包）**  
  以文件夹为单位组织，同一文件夹下所有 `.go` 文件属于同一个包。  
  一个模块可包含多个包。  
  包之间通过 import 导入，路径为 module 路径 + 子目录名。

- **GOROOT**  
  Go 安装目录（如 `/usr/local/go`），包含编译器、标准库等。  
  不建议修改，仅供内部使用。

- **GOPATH**  
  虽不再作为项目根目录，但仍用于：  
  - 缓存依赖模块：`$GOPATH/pkg/mod/`  
  - 安装工具包：`$GOPATH/bin/`（如 `go install` 安装的 CLI 工具）

---
## 项目结构与依赖行为示例
```
/home/user/myproject/ ← 当前项目根目录（模块根）
├── go.mod            ← 模块定义文件，声明依赖
├── go.sum            ← 依赖版本校验文件
├── main.go           ← 主程序入口，package main
├── service/          ← 自定义包，模块内代码组织
│   └── service.go
├── utils/
│   └── utils.go
└── vendor/ (可选)     ← 第三方依赖本地副本，使用 `go mod vendor` 生成

$GOPATH/pkg/mod/ ← 所有下载的第三方依赖模块统一缓存路径  
├─ github.com/gin-gonic/gin@v1.9.1/  
├─ golang.org/x/net@v0.0.0-xxxxxx/  
└─ ...  
```
- 每个子目录为一个 package  
- 同一个模块下的 package 可直接通过相对路径导入  
- 项目依赖的第三方库会被下载到 `$GOPATH/pkg/mod/`  
- 通过 `import "github.com/xxx/yyy"` 使用第三方包  
- 依赖关系自动写入 `go.mod`，完整校验写入 `go.sum`

---

## 常用命令说明

| 命令         | 功能                                      |
|--------------|-------------------------------------------|
| go mod init  | 初始化 go.mod 文件                        |
| go get       | 获取第三方依赖并写入 go.mod               |
| go build     | 构建项目（自动拉取依赖）                   |
| go run       | 编译并运行（自动拉取依赖）                 |
| go install   | 构建并安装到 `$GOPATH/bin`                 |
| go mod tidy  | 清理未用依赖，补全遗漏依赖                 |

---

## Go 关键路径及项目结构说明

- **项目目录**  
  包含 `go.mod` 的文件夹为模块根。模块内部可按功能划分多个包（package），包是 Go 的基本代码组织单元。项目代码与自定义包统一管理，灵活组织，支持独立版本控制。

- **GOPATH**  
  主要用于存放第三方依赖的模块缓存（`$GOPATH/pkg/mod`）。  
  无论项目位置如何，拉取的依赖会统一缓存在此路径下，多个模块共享缓存，避免重复下载。  
  通过 `go install` 安装的 CLI 工具默认放入 `$GOPATH/bin`。

- **GOROOT**  
  Go 安装目录，包含标准库源码及工具链（如 go、compile、link 等），一般不需要修改。

- **依赖管理机制**  
  `go.mod`：声明模块名、依赖模块及其版本（require），可通过 replace 替换路径。  
  `go.sum`：记录所有依赖模块的哈希校验值，确保依赖未被篡改，支持安全校验。  

---

## Go 项目依赖与构建流程

1. 安装 Go 语言环境（默认安装到 `/usr/local/go`），设置环境变量 `GOROOT` 与 `PATH`  
2. 使用 `go get` 获取依赖模块，支持模块之间相互依赖（包括第三方库）  
3. 使用 `go install` 安装可执行包，产物默认输出到 `$GOPATH/bin`  
4. 使用 `import` 引入模块内外部包，依赖统一由 `go.mod` 管理  
5. 同一 package 内，函数可跨文件直接使用；每个子文件夹即为一个 package  
6. `go.mod` 声明模块名、require 的依赖版本、replace 的本地路径替代项  
7. `go.sum` 记录所有依赖模块的哈希校验值，保障依赖一致与安全  
8. `go build`、`go test`、`go mod tidy` 等命令自动更新 `go.sum` 文件



# Goredis
## 一、渐进式 Rehash 方法
**目的**  
将旧哈希表中的数据逐步迁移到新表，避免一次性 rehash 带来的性能阻塞。

**关键机制**  
1. 字典结构有两个哈希表：`hts[0]`（当前表）和 `hts[1]`（新表）。  
2. 扩容时调用 `expandIfNeeded()` 触发 `expand(size)`，分配 `hts[1]` 并将 `rehashidx` 设为 0，标记开始渐进式迁移。  
3. 每次对字典进行操作（插入、查找、删除）时，会触发一步 `rehashStep()`，将 `hts[0].table[rehashidx]` 中的桶迁移到 `hts[1]`。  
4. 当 `hts[0]` 所有桶都迁移完毕（`rehashidx >= hts[0].size`），交换两个哈希表，并重置 `rehashidx = -1`。

**优点**  
将 rehash 分摊到日常操作中，避免长时间阻塞。

> 在进行插入、查找、删除等日常操作时，  
> 应首先判断字典是否正在进行 rehash（即 `dict.isRehashing() == true`），  
> 若是，则执行一步 `rehashStep()`，迁移当前 `rehashidx` 位置上的桶内容，  
> 从而将 rehash 操作与日常操作交错执行，实现渐进式迁移。

**rehash 期间的操作规则**  
- **查找（Find）**：同时查询 `hts[0]` 和 `hts[1]`，确保命中旧数据或已迁移的数据。  
- **插入（Add & Set）**：所有新键值对仅插入到 `hts[1]`。  
- **删除（Delete）**：需要在两个表中查找并删除目标键。  
- **过期**：本质也是一次删除再插入操作。  

这样设计确保了数据一致性，同时不会阻塞主线程，实现高性能的动态扩容。
```go
type Entry struct {
    Key  *Gobj
    Val  *Gobj
    next *Entry
}

type DictType struct {
    HashFunc  func(key *Gobj) int64
    EqualFunc func(k1, k2 *Gobj) bool
}

type Dict struct {
    DictType
    hts       [2]*htable
    rehashidx int64
}
...
func (dict *Dict) rehash(step int) {
	for step > 0 {
		if dict.hts[0].used == 0 {
			dict.hts[0] = dict.hts[1]
			dict.hts[1] = nil
			dict.rehashidx = -1
			return
		}
		// find an nonull slot
		for dict.hts[0].table[dict.rehashidx] == nil {
			dict.rehashidx += 1
		}
		// migrate all keys in this slot
		entry := dict.hts[0].table[dict.rehashidx]
		for entry != nil {
			ne := entry.next
			idx := dict.HashFunc(entry.Key) & dict.hts[1].mask
			entry.next = dict.hts[1].table[idx]
			dict.hts[1].table[idx] = entry
			dict.hts[0].used -= 1
			dict.hts[1].used += 1
			entry = ne
		}
		dict.hts[0].table[dict.rehashidx] = nil
		dict.rehashidx += 1
		step -= 1
	}
}
```

```go
type Node struct {
    Val  *Gobj
    next *Node
    prev *Node
}

type ListType struct {
    EqualFunc func(a, b *Gobj) bool
}

type List struct {
    ListType
    head   *Node
    tail   *Node
    length int
}

type GodisClient struct {
    ...
    args     []*Gobj
    reply    *List
    ...
}
```

## 二、写事件的非阻塞设计：List 与 reply 队列

对于每一次来自客户端的完整请求（如 `SET mykey hello`），在获取数据库数据（无论成功或失败）后，服务器都会调用 `AddReply`：

- 将响应结果封装成 `Node` 节点（注意：`Dict` 中的 `Gobj` 和 `List.Node` 中的 `Gobj` 是解耦合的）；
- 放入 `client.reply` 的 `List` 队列中；
- 调用 `loop.AddFileEvent(fd, AE_WRITABLE, SendReplyToClient, client)` 注册写事件；
  - 注册时会检查 `fd` 是否已经注册了 `AE_WRITABLE`，避免重复。

---

**对于 `List` 队列中的每一个 `Node`：**

**情况一：未全部写完**

- 由于内核缓冲区限制，调用 `unix.write(fd, buf[client.sentLen:])` 只写出了部分数据（即 `n < bufLen - sentLen`）；
- 增加 `client.sentLen`，记录已发送的字节数；
- **不删除当前回复节点**，保留剩余数据等待下一次发送；
- 退出发送循环，等待下一次写事件触发，继续发送。

**情况二：全部写完**

- 若 `client.sentLen == bufLen`，说明当前回复内容全部发送完成；
- 删除当前回复节点，释放资源：
  ```go
  client.reply.DelNode(rep)
  rep.Val.DecrRefCount()
  ```
- 重置 `client.sentLen = 0`，为下一条回复准备；
- 继续发送下一个节点（如果有）。

---

**状态处理总结**

| 情况           | 处理方式                                         | 目的                           |
|----------------|--------------------------------------------------|--------------------------------|
| **未全部写完** | 更新 `sentLen`，保留节点，等待下次写事件触发     | 处理短写，保证数据完整性和顺序 |
| **全部写完**   | 删除节点，清空进度，继续发送下一个               | 清理资源，推进回复队列         |

```go
type Node struct {
    Val  *Gobj
    next *Node
    prev *Node
}

type ListType struct {
    EqualFunc func(a, b *Gobj) bool
}

type List struct {
    ListType
    head   *Node
    tail   *Node
    length int
}

type GodisClient struct {
    ...
    args     []*Gobj
    reply    *List
    ...
}
...
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
...
func SendReplyToClient(loop *AeLoop, fd int, extra interface{}) {
	client := extra.(*GodisClient)
	log.Printf("SendReplyToClient, reply len:%v\n")   
  client.reply.Length()
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

```

## 三、读事件及 RESP 协议

### 客户端状态管理

- `freeClient`：彻底释放客户端资源，关闭连接。
  - 释放命令参数与回复队列；
  - 移除读写事件监听；
  - 从服务器客户端集合中删除；
  - 关闭文件描述符，完成断开。
  - 调用时机：出错或客户端发送 `quit` 命令。

- `resetClient`：重置客户端状态，保留连接与回复队列，仅释放命令参数。
  - 重置命令类型、bulk 状态；
  - 用于指令处理完成后的复用；
  - 正常流程：`ProcessCommand -> cmd.proc(c) -> resetClient(c)`。

---
```go
func freeClient(client *GodisClient) {
    freeArgs(client) 
    delete(server.clients, client.fd)
    server.aeLoop.RemoveFileEvent(client.fd, AE_READABLE)
    server.aeLoop.RemoveFileEvent(client.fd, AE_WRITABLE)
    freeReplyList(client)
    Close(client.fd)
}

func resetClient(client *GodisClient) {
    freeArgs(client)
    client.cmdTy = COMMAND_UNKNOWN
    client.bulkLen = 0
    client.bulkNum = 0
}
```

### 协议解析类型

| 函数                | 协议类型                   | 特点                                                                                          |
|---------------------|----------------------------|-----------------------------------------------------------------------------------------------|
| `handleInlineBuf`   | **Inline（行内）命令协议** | 命令和参数一整行发送，空格分隔。例如：`SET key value\r\n`                                    |
| `handleBulkBuf`     | **Bulk（多段）命令协议**   | Redis 使用的 RESP 协议，结构化明确，支持二进制数据。例如：`*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n` |

---

### 协议处理逻辑概述

#### Inline 协议（handleInlineBuf）

```go
index, err := client.findLineInQuery()
if index < 0 {
    return false, err
}
```

- 查找 `\r\n`，如果未找到，说明命令未读完，提前返回；
- 否则继续分割参数，构建 `args` 数组。

---

#### Bulk 协议（handleBulkBuf）以及中断逻辑
```
RESP 协议处理代码
findLineInQuery() 寻找 \r\n
getNumInQuery(1, index) 获取数字
均包含了边界处理的逻辑
分为有无 bulkNum 的情况:
有的话说明因为没有完整命令已经 break 过一次
没有则应当从 queryBuf 获取 bulkNum：
```
##### 1. 获取参数数量（*N\r\n）

```go
index, err := client.findLineInQuery()
if index < 0 {
    return false, err
}
bnum, err := client.getNumInQuery(1, index)
client.bulkNum = bnum
```

##### 2. 读取每个 bulk 长度（$len\r\n）

```go
if client.bulkLen == 0 {
    index, err := client.findLineInQuery()
    if index < 0 {
        return false, err // 仍然不完整
    }
    if client.queryBuf[0] != '$' {
        return false, errors.New("expect $ for bulk length")
    }
    blen, err := client.getNumInQuery(1, index)
    ...
    client.bulkLen = blen
}
```

##### 3. 读取 bulk 内容（数据本体）

```go
if client.queryLen < client.bulkLen + 2 {
    return false, nil // 内容未完整，等待更多数据
}
if client.queryBuf[index] != '\r' || client.queryBuf[index+1] != '\n' {
    return false, errors.New("expect CRLF for bulk end")
}
client.args[len(client.args)-client.bulkNum] = CreateObject(GSTR, string(client.queryBuf[:index]))
client.queryBuf = client.queryBuf[index+2:]
client.queryLen -= index + 2
client.bulkLen = 0
client.bulkNum -= 1
```

---

### 中断与继续

- 外层 `ReadQueryFromClient` 会根据 `false` 返回值决定是否继续等待数据；
- 协议处理函数只负责判断当前是否接收完整，不进行跳转；
- 一旦 `args` 填充完成，即进入命令执行阶段；
  ```go
  ProcessCommand -> cmd.proc(c)
  ```

- 数据处理后，调用：
  - `AddReplyStr`：构造响应；
  - 注册 `AE_WRITABLE` 写事件；
  - 将响应加入 `client.reply` 的 `List` 队列。

---

### 总结流程图（文字版）

1. 接收请求 → 判断协议类型（Inline 或 Bulk）
2. 解析命令参数（支持多次中断恢复）
3. 构建 Gobj 并存入 args
4. 命令执行，得到结果
5. AddReply，加入写队列，注册写事件
6. 等待事件驱动发送回复

## 四、单线程 EventLoop 模型

- **连接建立：**  
  `AcceptHandler` 负责建立连接，封装 `*GodisClient` 对象，添加到维护的 `Clients` 队列（`[]*GodisClient`）。

- **事件注册：**  
  - 对于 **FileEvent**，有两次注册：  
    调用 `AddFileEvent` 注册读事件（内核态调用 `epoll_ctl`，用户态维护 `map[int]*aeFileEvent`）。  
  - 对于 **TimeEvent**，初始化 `aeTimeEvent` 并设置触发时间 `when`，采用头插法加入 `[]*aeTimeEvent`。

- **事件获取与调度：**  
  - `AeWait` 获取返回的文件描述符和事件类型，遍历后通过 `map[int]*aeFileEvent` 查询对应事件，放入事件队列 `[]*aeFileEvent`。  
  - 遍历 `[]*aeTimeEvent`，将满足触发条件的时间事件加入队列。

- **事件执行：**  
  - `aeFileEvent` 调用其 `proc(*, fd, extra)` 函数处理。  
  - `aeTimeEvent` 调用前检查 `mask` 是否需要一次性清除；调用后重新设置触发时间（`+= interval`）。

---

### 随机过期清理（ServerCron）

- 定期从过期字典随机抽取若干键，检查是否过期。  
- 发现过期则从主数据和过期字典中删除。  
- 这是当前唯一的 `TimeEvent`。

```go
func ServerCron(loop *AeLoop, id int, extra interface{}) {
	for i := 0; i < EXPIRE_CHECK_COUNT; i++ {
		entry := server.db.expire.RandomGet()
		if entry == nil {
			break
		}
		if entry.Val.IntVal() < time.Now().Unix() {
			server.db.data.Delete(entry.Key)
			server.db.expire.Delete(entry.Key)
		}
	}
}
```
### 总体单线程模型流程
- 单线程	所有操作在同一线程完成，避免锁竞争，依赖事件循环实现高并发。
- 非阻塞 I/O	通过 epoll 实现多路复用，文件事件与定时事件统一调度，减少上下文切换。
```go
func main() {
	...
	// 文件事件注册，监听可读事件，处理新连接
	server.aeLoop.AddFileEvent(server.fd, AE_READABLE, AcceptHandler, nil)

	// 时间事件注册，作为后台定时任务执行过期清理
	server.aeLoop.AddTimeEvent(AE_NORMAL, 100, ServerCron, nil) 

	log.Println("godis server is up.")
	// 启动事件循环主程序
	server.aeLoop.AeMain()
}
```
### 五、关于 *Gobj 的 reference int64

- **ref 概念：**  
  `ref` 实际是一个可视化的标记，内存管理由 Go 语言自动完成释放。

- **对于 Dict 来说：**  
  - 从 `CreateObj` 到存入 `args[]` 时，`ref = 1`。  
  - 后续对数据库的写操作会改变其中的值（GET 操作不会）。  
  - 一旦 `*Gobj` 存入 Dict，`ref` 保持为 1。  
  - 客户端在 `freeclient` 或 `resetclient` 时，会调用 `freeargs()`，对所有 `ref` 执行递减。

- **对于 Reply 来说：**  
  - `List` 与 `Dict` 中的 `*Gobj` 是解耦的，Reply 中的对象是从数据库取值后重新封装的。  
  - `CreateObj` 初始化时，`ref = 1`。  
  - `AddReply` 将 `*Gobj` 封装成节点（Node）时会增加引用计数；而 `AddReplyStr` 会相应释放一次，确保节点删除前，`*Gobj` 仅有一次引用存在于 `List` 中。  
  - 发送完成后，释放节点时，会删除头结点并递减引用计数：

```go
rep := client.reply.First()
...
if client.sentLen == bufLen {
	client.reply.DelNode(rep)
	rep.Val.DecrRefCount()
	client.sentLen = 0
}
```
## 特别鸣谢
### bilibili-楚国吹大风