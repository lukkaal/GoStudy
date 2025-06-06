package main

import (
	"context" // 新增: 用于 Etcd 操作的上下文
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"strconv" // 新增: 用于端口转换为字符串
	"sync"
	"syscall"
	"time"

	"gorpc/service"

	clientv3 "go.etcd.io/etcd/client/v3" // 新增: Etcd 客户端库
)

const (
	// RPCServerHost 服务器的 IP 地址，如果是多网卡或者对外服务，需要配置实际IP
	RPCServerHost = "127.0.0.1"
	// RPCServerPort 服务器监听的端口
	RPCServerPort = 8080

	// GracefulShutdownTimeout 定义优雅关闭的超时时间
	GracefulShutdownTimeout = 5 * time.Second

	// --- Etcd 相关新增/修改 开始 ---
	// EtcdEndpoints Etcd 集群的地址，多个用逗号分隔
	EtcdEndpoints = "127.0.0.1:2389"
	// EtcdServicePrefix Etcd 中服务注册的键前缀
	EtcdServicePrefix = "/myrpc/services/UserService/" // 例如：/myrpc/services/UserService/127.0.0.1:8080
	// EtcdLeaseTTL Etcd 租约的过期时间（秒），客户端会据此判断服务是否存活
	EtcdLeaseTTL = 10
	// --- Etcd 相关新增/修改 结束 ---
)

func main() {
	// 1. 创建 UserService 实例
	userService := service.NewUserService()

	// 2. 将服务注册到 RPC 服务器
	err := rpc.Register(userService)
	if err != nil {
		log.Fatalf("Failed to register RPC service: %v", err)
	}
	fmt.Println("RPC service 'UserService' registered successfully.")

	// 构建服务器监听地址
	rpcServerAddress := net.JoinHostPort(RPCServerHost, strconv.Itoa(RPCServerPort)) // 使用 strconv.Itoa 转换端口
	listener, err := net.Listen("tcp", rpcServerAddress)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", rpcServerAddress, err)
	}
	fmt.Printf("RPC server listening on %s...\n", rpcServerAddress)

	// --- Etcd 相关新增/修改 开始 ---
	// 初始化 Etcd 客户端
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{EtcdEndpoints},
		DialTimeout: 5 * time.Second, // 连接 Etcd 的超时时间
	})
	if err != nil {
		// Fatalf is equivalent to [Printf] followed by a call to os.Exit(1) 执行退出
		// log.SetOutput(f) 设置后可以打印到文件 否则默认打印到终端
		log.Fatalf("Failed to connect to Etcd: %v", err)
	}
	defer func() {
		fmt.Println("Closing Etcd client connection...")
		if err := etcdClient.Close(); err != nil {
			log.Printf("Error closing Etcd client: %v", err)
		}
	}()

	// 用于 Etcd 操作的 Context，控制租约和心跳的生命周期
	etcdCtx, etcdCancel := context.WithCancel(context.Background())
	defer etcdCancel() // 确保 Etcd Context 在 main 函数结束时被取消

	// 获取租约
	leaseResp, err := etcdClient.Grant(etcdCtx, EtcdLeaseTTL)
	if err != nil {
		log.Fatalf("Failed to grant Etcd lease: %v", err)
	}
	leaseID := leaseResp.ID
	fmt.Printf("Etcd lease granted with ID: %x, TTL: %d seconds\n", leaseID, EtcdLeaseTTL)

	// 构建注册到 Etcd 的键值对
	// 键: /myrpc/services/UserService/127.0.0.1:8080
	// 值: 127.0.0.1:8080 (可以包含更多元数据，如权重、版本等)
	serviceKey := EtcdServicePrefix + rpcServerAddress
	serviceValue := rpcServerAddress

	// 将服务信息注册到 Etcd，并绑定租约
	_, err = etcdClient.Put(etcdCtx, serviceKey, serviceValue, clientv3.WithLease(leaseID))
	if err != nil {
		log.Fatalf("Failed to register service in Etcd: %v", err)
	}
	fmt.Printf("Service registered in Etcd: Key=%s, Value=%s\n", serviceKey, serviceValue)

	// 保持心跳：启动一个 goroutine 定期续租
	// Etcd 客户端库的 KeepAlive 会自动发送心跳
	keepAliveChan, err := etcdClient.KeepAlive(etcdCtx, leaseID)
	if err != nil {
		log.Fatalf("Failed to start Etcd keep-alive: %v", err)
	}
	go func() {
		for {
			select {
			case kaResp := <-keepAliveChan:
				if kaResp == nil {
					fmt.Println("Etcd keep-alive channel closed. Lease expired or Etcd connection lost. Exiting server.")
					// 如果心跳通道关闭，通常意味着与Etcd的连接出现问题，
					// 或者租约过期未续，此时服务应该考虑下线

					// 用于立即终止当前程序（服务端）的运行，并返回指定的退出码给操作系统
					os.Exit(1) // 简单粗暴的退出，生产中可能需要重连或更复杂的处理
					return
				}
				// fmt.Printf("Etcd keep-alive received: TTL %d\n", kaResp.TTL) // 如果想看心跳信息可以取消注释
			case <-etcdCtx.Done(): // Etcd context 被取消时退出
				fmt.Println("Etcd keep-alive goroutine shutting down due to context cancellation.")
				return
			}
		}
	}()
	fmt.Println("Etcd keep-alive goroutine started for lease.")
	// --- Etcd 相关新增/修改 结束 ---

	// --- 优雅退出逻辑 (与之前相同，但需要处理 Etcd 注销) ---
	var wg sync.WaitGroup
	shutdownAcceptChan := make(chan struct{}) // 通知 Accept 循环停止接受新连接
	sigChan := make(chan os.Signal, 1)        // 接收操作系统信号
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动一个 goroutine 来接受连接
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-shutdownAcceptChan: // 收到关闭信号，且 Accept 失败，表示监听器已关闭，正常退出循环
					fmt.Println("Listener stopped accepting new connections.")
					return
				default: // 其他错误，可能是网络问题，短暂休眠后重试
					log.Printf("Failed to accept connection: %v", err)
					time.Sleep(time.Second)
					continue
				}
			}
			fmt.Printf("Accepted new connection from %s\n", conn.RemoteAddr())

			wg.Add(1) // 增加一个等待计数
			go func() {
				defer wg.Done()    // 在 goroutine 结束时减少计数
				defer conn.Close() // 确保连接被关闭

				rpc.ServeConn(conn) // 阻塞
				fmt.Printf("Connection from %s handled and closed.\n", conn.RemoteAddr())
			}()
		}
	}()

	// 监听信号并处理优雅退出
	sig := <-sigChan
	fmt.Printf("Received signal %v. Initiating graceful shutdown...\n", sig)

	// 1. 停止接受新连接
	// close 会向管道当中传入信号量 (empty)
	close(shutdownAcceptChan) // 通知 Accept 循环退出
	listener.Close()          // 关闭监听器，导致 listener.Accept() 返回错误并退出循环

	// --- Etcd 相关新增/修改 开始 ---
	// 2. 优雅地从 Etcd 注销服务
	fmt.Printf("Attempting to unregister service from Etcd: %s\n", serviceKey)
	// 使用新的 context，避免被 main 函数或 Etcd keep-alive 的 context 影响
	unregisterCtx, unregisterCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer unregisterCancel()

	// 先删除键
	_, err = etcdClient.Delete(unregisterCtx, serviceKey)
	if err != nil {
		log.Printf("Failed to unregister service from Etcd (delete key): %v", err)
	} else {
		fmt.Printf("Service key deleted from Etcd: %s\n", serviceKey)
	}

	// 如果服务当初是通过租约注册的（即 leaseID 有效）
	if leaseID != clientv3.NoLease {
		fmt.Printf("Revoking Etcd lease: %x\n", leaseID)
		// 然后撤销租约，强制 Etcd 移除所有相关键
		_, err := etcdClient.Revoke(unregisterCtx, leaseID)
		if err != nil {
			log.Printf("Failed to revoke Etcd lease: %v", err)
		}
	}
	etcdCancel() // 取消 Etcd 客户端操作的 context，这将停止控制租约和心跳的生命周期
	// --- Etcd 相关新增/修改 结束 ---

	// 3. 等待所有活跃的 RPC 调用完成，但有超时
	done := make(chan struct{})
	go func() {
		wg.Wait() // 等待所有 rpc.ServeConn goroutine 完成
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("All active connections handled. Server shut down gracefully.")
	case <-time.After(GracefulShutdownTimeout):
		fmt.Printf("Graceful shutdown timed out after %s. Forcing shutdown.\n", GracefulShutdownTimeout)
	}

	fmt.Println("Server exited.")
}
