package main

import (
	"context"
	"log"
	"time"

	pb "protogorpc/api" // 导入生成的 Protobuf 包

	"google.golang.org/grpc"                      // 导入 GRPC 核心库
	"google.golang.org/grpc/credentials/insecure" // 用于不使用TLS的连接
)

const (
	serverAddr  = "localhost:8080" // 硬编码服务器地址
	defaultName = "World"
)

func main() {
	// 建立与 GRPC 服务端的连接
	// grpc.WithTransportCredentials(insecure.NewCredentials()) 表示不使用 TLS/SSL，
	// 这在开发环境中方便，但生产环境强烈推荐使用 TLS。
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close() // 确保连接在使用完毕后关闭

	// 创建 UserService 的客户端存根
	c := pb.NewUserServiceClient(conn)

	// 设置一个带超时的上下文，用于控制 RPC 请求的生命周期
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel() // 确保上下文在操作完成后被取消，释放资源

	// --- 调用 CreateUser 方法 ---
	log.Println("--- Calling CreateUser ---")
	createReq := &pb.CreateUserRequest{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   30,
	}
	createResp, err := c.CreateUser(ctx, createReq) // 发起 RPC 调用
	if err != nil {
		log.Fatalf("could not create user: %v", err)
	}
	log.Printf("CreateUser response: ID=%s, Message=%s", createResp.GetId(), createResp.GetMessage())

	// --- 调用 GetUser 方法 ---
	log.Println("--- Calling GetUser ---")
	userIDToGet := createResp.GetId() // 使用刚刚创建的用户ID来获取
	getReq := &pb.GetUserRequest{
		Id: userIDToGet,
	}
	getResp, err := c.GetUser(ctx, getReq) // 发起 RPC 调用
	if err != nil {
		log.Fatalf("could not get user: %v", err)
	}
	log.Printf("GetUser response: ID=%s, Name=%s, Email=%s, Age=%d",
		getResp.GetId(), getResp.GetName(), getResp.GetEmail(), getResp.GetAge())

	// --- 尝试获取一个不存在的用户 ---
	log.Println("--- Calling GetUser for non-existent ID ---")
	nonExistentUserID := "non-existent-user-123"
	getNonExistentReq := &pb.GetUserRequest{
		Id: nonExistentUserID,
	}
	_, err = c.GetUser(ctx, getNonExistentReq)
	if err != nil {
		log.Printf("Successfully failed to get non-existent user (expected): %v", err)
	} else {
		log.Println("Unexpectedly found non-existent user.")
	}

}
