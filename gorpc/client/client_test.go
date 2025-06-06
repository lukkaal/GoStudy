package main

import (
	"fmt"
	"log"
	"net/rpc" // 导入 Go 官方的 RPC 包
	"time"

	// 导入我们定义的服务包，因为客户端需要知道请求和响应的结构体定义
	"gorpc/service"
)

const (
	// 与服务器端监听地址保持一致
	RPCServerAddress = "127.0.0.1:8080"
)

func initclient_test() {
	// 1. 连接 RPC 服务器
	// rpc.Dial 可以建立一个 TCP 连接，并为 RPC 通信做准备
	client, err := rpc.Dial("tcp", RPCServerAddress)
	if err != nil {
		log.Fatalf("Failed to dial RPC server: %v", err)
	}
	defer func() {
		fmt.Println("Closing RPC client connection...")
		if err := client.Close(); err != nil {
			log.Printf("Error closing RPC client: %v", err)
		}
	}()
	fmt.Printf("Connected to RPC server at %s\n", RPCServerAddress)

	// --- 2. 调用 CreateUser 方法 ---
	fmt.Println("\n--- Calling CreateUser ---")
	createReq := service.CreateUserRequest{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   30,
	}
	var createResp service.CreateUserResponse
	// rpc.Call(serviceMethod string, args interface{}, reply interface{}) error
	// serviceMethod: "服务名.方法名"
	err = client.Call("UserService.CreateUser", createReq, &createResp)
	if err != nil {
		log.Fatalf("Failed to call CreateUser: %v", err)
	}
	if createResp.Error != "" {
		fmt.Printf("CreateUser business error: %s\n", createResp.Error)
	} else {
		fmt.Printf("User created: %+v\n", createResp.User)
	}
	// 保存创建的用户ID，以便后续获取和更新
	createdUserID := createResp.User.ID

	// --- 3. 调用 GetUser 方法 ---
	fmt.Println("\n--- Calling GetUser (existing user) ---")
	getReq := service.GetUserRequest{
		ID: createdUserID,
	}
	var getResp service.GetUserResponse
	err = client.Call("UserService.GetUser", getReq, &getResp)
	if err != nil {
		log.Fatalf("Failed to call GetUser: %v", err)
	}
	if getResp.Error != "" {
		fmt.Printf("GetUser business error: %s\n", getResp.Error)
	} else {
		fmt.Printf("User retrieved: %+v\n", getResp.User)
	}

	fmt.Println("\n--- Calling GetUser (non-existent user) ---")
	getNonExistentReq := service.GetUserRequest{
		ID: "non-existent-user-123",
	}
	var getNonExistentResp service.GetUserResponse
	err = client.Call("UserService.GetUser", getNonExistentReq, &getNonExistentResp)
	if err != nil {
		log.Fatalf("Failed to call GetUser for non-existent user: %v", err)
	}
	if getNonExistentResp.Error != "" {
		fmt.Printf("GetUser business error: %s\n", getNonExistentResp.Error)
	} else {
		fmt.Printf("User retrieved (should not happen for non-existent): %+v\n", getNonExistentResp.User)
	}

	// --- 4. 调用 UpdateUser 方法 ---
	fmt.Println("\n--- Calling UpdateUser ---")
	updateReq := service.UpdateUserRequest{
		ID:    createdUserID,
		Name:  "Alice Updated",
		Email: "alice.updated@example.com",
		Age:   31,
	}
	var updateResp service.UpdateUserResponse
	err = client.Call("UserService.UpdateUser", updateReq, &updateResp)
	if err != nil {
		log.Fatalf("Failed to call UpdateUser: %v", err)
	}
	if updateResp.Error != "" {
		fmt.Printf("UpdateUser business error: %s\n", updateResp.Error)
	} else {
		fmt.Printf("User updated: %+v\n", updateResp.User)
	}
	// 再次获取用户验证更新
	fmt.Println("\n--- Calling GetUser (after update) ---")
	getReqAfterUpdate := service.GetUserRequest{
		ID: createdUserID,
	}
	var getRespAfterUpdate service.GetUserResponse
	err = client.Call("UserService.GetUser", getReqAfterUpdate, &getRespAfterUpdate)
	if err != nil {
		log.Fatalf("Failed to call GetUser after update: %v", err)
	}
	if getRespAfterUpdate.Error != "" {
		fmt.Printf("GetUser business error: %s\n", getRespAfterUpdate.Error)
	} else {
		fmt.Printf("User retrieved after update: %+v\n", getRespAfterUpdate.User)
	}

	fmt.Println("\nAll RPC calls completed.")
	time.Sleep(time.Second) // 留点时间观察输出
}
