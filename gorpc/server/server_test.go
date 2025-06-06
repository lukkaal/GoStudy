package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	// 导入 Etcd 客户端库
	// 导入我们定义的服务包
	"gorpc/service"
)

const (
	RPCServerAddress = "127.0.0.1:8080"
	// GracefulShutdownTimeout 定义优雅关闭的超时时间
	// GracefulShutdownTimeout = 5 * time.Second
)

func initserver_test() {
	userService := service.NewUserService()
	err := rpc.Register(userService)
	if err != nil {
		log.Fatalf("Failed to register RPC service: %v", err)
	}
	fmt.Println("RPC service 'UserService' registered successfully.")

	listener, err := net.Listen("tcp", RPCServerAddress)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", RPCServerAddress, err)
	}
	// 在程序退出时关闭监听器，但这不会等待现有连接
	// defer func() {
	// 	fmt.Println("Closing listener...")
	// 	if err := listener.Close(); err != nil {
	// 		log.Printf("Error closing listener: %v", err)
	// 	}
	// }()

	fmt.Printf("RPC server listening on %s...\n", RPCServerAddress)

	// 用于等待所有处理连接的 goroutine 完成
	var wg sync.WaitGroup
	// 用于通知 Accept 循环停止接受新连接
	shutdownChan := make(chan struct{})
	// 用于接收操作系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM) // 监听 Ctrl+C 和 kill 命令

	// 启动一个 goroutine 来接受连接
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-shutdownChan:
					// 如果收到了关闭信号，并且 Accept 失败，可能是监听器已关闭，正常退出循环
					fmt.Println("Listener stopped accepting new connections.")
					return
				default:
					log.Printf("Failed to accept connection: %v", err)
					time.Sleep(time.Second) // 短暂休眠后重试
					continue
				}
			}
			fmt.Printf("Accepted new connection from %s\n", conn.RemoteAddr())

			wg.Add(1) // 增加一个等待计数
			go func() {
				defer wg.Done()    // 在 goroutine 结束时减少计数
				defer conn.Close() // 确保连接被关闭

				rpc.ServeConn(conn)
				fmt.Printf("Connection from %s handled and closed.\n", conn.RemoteAddr())
			}()
		}
	}()

	// 监听信号并处理优雅退出
	sig := <-sigChan
	fmt.Printf("Received signal %v. Initiating graceful shutdown...\n", sig)

	// 1. 停止接受新连接
	close(shutdownChan) // 关闭通道，通知 Accept 循环退出
	listener.Close()    // 关闭监听器，导致 listener.Accept() 返回错误并退出循环

	// 2. 等待所有活跃的 RPC 调用完成，但有超时
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
