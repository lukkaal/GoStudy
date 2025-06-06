package service

import (
	"fmt"
	"sync"
)

// User 定义了用户结构体
type User struct {
	ID    string
	Name  string
	Email string
	Age   int
}

// CreateUserRequest 定义创建用户方法的请求参数
type CreateUserRequest struct {
	Name  string
	Email string
	Age   int
}

// CreateUserResponse 定义创建用户方法的响应
type CreateUserResponse struct {
	User  *User
	Error string
}

// GetUserRequest 定义获取用户方法的请求参数
type GetUserRequest struct {
	ID string
}

// GetUserResponse 定义获取用户方法的响应
type GetUserResponse struct {
	User  *User
	Error string
}

// UpdateUserRequest 定义更新用户方法的请求参数
type UpdateUserRequest struct {
	ID    string
	Name  string
	Email string
	Age   int
}

// UpdateUserResponse 定义更新用户方法的响应
type UpdateUserResponse struct {
	User  *User
	Error string
}

// UserService 是一个用户管理的 RPC 服务
type UserService struct {
	// 简单的内存存储，实际应用中会是数据库
	users  map[string]*User
	mu     sync.RWMutex // 读写锁，保证并发安全
	nextID int          // 用于生成简易的用户ID
}

// NewUserService 创建并返回一个新的 UserService 实例
func NewUserService() *UserService {
	return &UserService{
		users:  make(map[string]*User),
		nextID: 1,
	}
}

// CreateUser 方法用于创建新用户
func (s *UserService) CreateUser(req *CreateUserRequest, resp *CreateUserResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 简单的ID生成
	newID := fmt.Sprintf("user-%d", s.nextID)
	s.nextID++

	user := &User{
		ID:    newID,
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
	}
	s.users[newID] = user

	resp.User = user
	resp.Error = ""
	fmt.Printf("Created user: %+v\n", user)
	return nil
}

// GetUser 方法根据用户ID获取用户信息
func (s *UserService) GetUser(req *GetUserRequest, resp *GetUserResponse) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[req.ID]
	if !ok {
		resp.User = nil
		resp.Error = fmt.Sprintf("User with ID %s not found", req.ID)
		fmt.Printf("Get user failed: %s\n", resp.Error)
		// 注意：RPC 方法返回 error 表示通信或内部逻辑错误，
		// 业务逻辑错误（如用户不存在）通常放在响应结构体中
		return nil
	}

	resp.User = user
	resp.Error = ""
	fmt.Printf("Retrieved user: %+v\n", user)
	return nil
}

// UpdateUser 方法更新现有用户信息
func (s *UserService) UpdateUser(req *UpdateUserRequest, resp *UpdateUserResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[req.ID]
	if !ok {
		resp.User = nil
		resp.Error = fmt.Sprintf("User with ID %s not found for update", req.ID)
		fmt.Printf("Update user failed: %s\n", resp.Error)
		return nil
	}

	// 更新字段
	user.Name = req.Name
	user.Email = req.Email
	user.Age = req.Age

	resp.User = user
	resp.Error = ""
	fmt.Printf("Updated user: %+v\n", user)
	return nil
}
