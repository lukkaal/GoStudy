package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync" // 用于保护用户数据
	"syscall"

	pb "protogorpc/api" // 导入生成的 Protobuf 包

	"google.golang.org/grpc" // 导入 GRPC 核心库
)

const (
	port = ":8080" // GRPC 服务器监听端口
)

// User 定义用户模型（为了简化，这里直接在内存中存储）
type User struct {
	ID    string
	Name  string
	Email string
	Age   int32
}

// server 是我们 GRPC User Service 的实现
type server struct {
	pb.UnimplementedUserServiceServer                  // 嵌入这个是为了向前兼容，即使未来 .proto 文件新增方法，旧代码也不会报错
	users                             map[string]*User // 内存中的用户存储
	mu                                sync.RWMutex     // 读写锁，保护 users map
	nextUserID                        int              // 下一个用户ID的计数器
}

// NewServer 创建并返回一个新的 GRPC 服务实例
func NewServer() *server {
	return &server{
		users:      make(map[string]*User),
		nextUserID: 1,
	}
}

// CreateUser 实现了 UserServiceServer 接口的 CreateUser 方法
func (s *server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	s.mu.Lock() // 写锁定，因为我们要修改 map
	defer s.mu.Unlock()

	userID := "user-" + strconv.Itoa(s.nextUserID)
	s.nextUserID++

	user := &User{
		ID:    userID,
		Name:  req.GetName(),
		Email: req.GetEmail(),
		Age:   req.GetAge(),
	}
	s.users[userID] = user // 存储用户

	log.Printf("Created user: ID=%s, Name=%s, Email=%s, Age=%d", user.ID, user.Name, user.Email, user.Age)

	return &pb.CreateUserResponse{
		Id:      user.ID,
		Message: "User created successfully",
	}, nil
}

// GetUser 实现了 UserServiceServer 接口的 GetUser 方法
func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	s.mu.RLock() // 读锁定，因为我们只读取 map
	defer s.mu.RUnlock()

	user, ok := s.users[req.GetId()]
	if !ok {
		log.Printf("User with ID %s not found", req.GetId())
		// GRPC 错误处理：使用 status 包返回特定的 GRPC 错误码
		// 例如，NotFound，而不是直接返回Go的 error 类型
		return nil, fmt.Errorf("user with ID %s not found", req.GetId()) // 简化处理，实际应使用 status.Errorf(codes.NotFound, ...)
	}

	log.Printf("Retrieved user: ID=%s, Name=%s", user.ID, user.Name)

	return &pb.GetUserResponse{
		Id:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Age:   user.Age,
	}, nil
}

func main() {
	// 1. 创建监听器和GRPC服务器
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", port, err)
	}
	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, NewServer())

	// 2. 在一个**独立的 Goroutine** 中启动 GRPC 服务器
	//    这样 main Goroutine 就可以继续执行后续代码（例如设置信号监听）。
	go func() {
		log.Printf("GRPC server listening on %s", port)
		if err := s.Serve(lis); err != nil { // 它会阻塞在那里，直到服务器被优雅地停止或发生错误。
			// 如果是 GracefulStop 导致的错误，这里会捕获到
			log.Printf("GRPC server stopped: %v", err)
		}
	}()

	// 3. 在 **main Goroutine** 中设置信号监听
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 4. main Goroutine 阻塞在这里，等待关闭信号
	//    这确保了 main Goroutine 不会提前退出，从而允许服务器 Goroutine 持续运行。
	<-quit
	log.Println("Shutting down GRPC server gracefully...")

	// 5. 收到信号后，调用 GracefulStop()
	//    这会使得 s.Serve(lis) 在其 Goroutine 中返回
	s.GracefulStop()

	log.Println("GRPC server exited.")
}
