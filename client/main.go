package main

import (
	"context" // 新增: 用于 Etcd 操作的上下文
	"errors"  // 新增: 用于返回错误
	"fmt"
	"log"
	"math/rand" // 新增: 用于负载均衡 (随机选择服务地址)
	"net"       // 新增: 用于 net.DialTimeout
	"net/rpc"   // 新增: 用于字符串操作 (如果需要解析Etcd键)
	"sync"      // 新增: 用于并发安全 (读写锁)
	"time"      // 新增: 用于时间操作

	"gorpc/service"

	clientv3 "go.etcd.io/etcd/client/v3" // 新增: Etcd 客户端库
)

const (
	// 原来的 RPCServerAddress 常量不再需要硬编码，因为会从 Etcd 发现

	// --- Etcd 相关新增/修改 开始 ---
	// EtcdEndpoints Etcd 集群的地址
	EtcdEndpoints = "127.0.0.1:2389"
	// EtcdServicePrefix Etcd 中服务注册的键前缀 (与服务器端一致)
	EtcdServicePrefix = "/myrpc/services/UserService/"

	// DialTimeout 连接 RPC 服务器的超时时间
	DialTimeout = 3 * time.Second
	// DiscoverTimeout 发现服务的超时时间
	DiscoverTimeout = 5 * time.Second
	// --- Etcd 相关新增/修改 结束 ---
)

// --- Etcd 相关新增/修改 开始 ---
// serviceDiscovery 结构体用于服务发现
type serviceDiscovery struct {
	etcdClient       *clientv3.Client
	serviceAddresses []string      // 存储发现到的服务地址列表
	mu               sync.RWMutex  // 读写锁保护 serviceAddresses
	stopWatchChan    chan struct{} // 用于通知监听 Etcd 变化的 goroutine 停止
}

// NewServiceDiscovery 创建并初始化服务发现客户端
func NewServiceDiscovery(endpoints []string, servicePrefix string) (*serviceDiscovery, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second, // 连接 Etcd 的超时
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %v", err)
	}

	sd := &serviceDiscovery{
		etcdClient:       cli,
		serviceAddresses: make([]string, 0),
		stopWatchChan:    make(chan struct{}),
	}

	// 首次从 Etcd 加载服务列表
	err = sd.loadServices(servicePrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to load services from etcd: %v", err)
	}

	// 启动一个 goroutine 监听 Etcd 变化
	go sd.watchServices(servicePrefix)

	return sd, nil
}

// Close 关闭 Etcd 客户端连接和服务发现的监听
func (sd *serviceDiscovery) Close() error {
	close(sd.stopWatchChan) // 停止监听 goroutine
	return sd.etcdClient.Close()
}

// loadServices 从 Etcd 加载所有可用的服务地址
func (sd *serviceDiscovery) loadServices(servicePrefix string) error {
	ctx, cancel := context.WithTimeout(context.Background(), DiscoverTimeout)
	defer cancel()

	// Get 带有前缀的所有键值对
	resp, err := sd.etcdClient.Get(ctx, servicePrefix, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("failed to get services with prefix %s: %v", servicePrefix, err)
	}

	var newAddresses []string
	for _, ev := range resp.Kvs {
		addr := string(ev.Value)
		newAddresses = append(newAddresses, addr)
		fmt.Printf("Discovered service on startup: %s -> %s\n", string(ev.Key), addr)
	}
	// 加锁
	/*
		主线程在调用 GetRandomServiceAddress 读取服务地址。
		后台的 watch goroutine 在监听 Etcd 变化时，可能会调用 addService、removeService
		或 loadServices 来修改服务地址列表。
	*/
	sd.mu.Lock()
	sd.serviceAddresses = newAddresses
	sd.mu.Unlock()
	fmt.Printf("Loaded %d services from Etcd on startup.\n", len(newAddresses))
	return nil
}

// watchServices 监听 Etcd 中服务键的变化 (新增、删除、修改)
func (sd *serviceDiscovery) watchServices(servicePrefix string) {
	// WatchCtx 返回一个 WatchChan，当 context 被取消时 WatchChan 会关闭
	rch := sd.etcdClient.Watch(context.Background(), servicePrefix, clientv3.WithPrefix())
	fmt.Printf("Watching Etcd for service changes on prefix: %s\n", servicePrefix)

	for {
		select {
		case resp, ok := <-rch:
			if !ok { // Watch channel closed, potentially due to Etcd connection issue
				fmt.Println("Etcd watch channel closed, re-establishing watch after 1 second...")
				time.Sleep(time.Second) // Wait a bit before retrying watch
				rch = sd.etcdClient.Watch(context.Background(), servicePrefix, clientv3.WithPrefix())
				continue
			}
			if resp.Canceled { // Watch context was canceled
				fmt.Printf("Etcd watch was canceled: %v\n", resp.Err())
				return // Exit goroutine
			}
			if resp.Err() != nil { // Other watch error
				fmt.Printf("Etcd watch error: %v\n", resp.Err())
				// Consider reconnecting or exiting based on error severity
				time.Sleep(time.Second)
				continue
			}

			for _, ev := range resp.Events {
				switch ev.Type {
				case clientv3.EventTypePut: // 新增或修改服务
					addr := string(ev.Kv.Value)
					sd.addService(addr)
					fmt.Printf("Service added/updated via watch: %s -> %s\n", string(ev.Kv.Key), addr)
				case clientv3.EventTypeDelete: // 删除服务
					addr := string(ev.PrevKv.Value) // PrevKv 包含删除前的键值对
					sd.removeService(addr)
					fmt.Printf("Service removed via watch: %s -> %s\n", string(ev.PrevKv.Key), addr)
				}
			}
		case <-sd.stopWatchChan: // 收到停止信号
			fmt.Println("Service discovery watch goroutine stopping.")
			return
		}
	}
}

// addService 将服务地址添加到列表中
func (sd *serviceDiscovery) addService(addr string) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	found := false
	for _, existingAddr := range sd.serviceAddresses {
		if existingAddr == addr {
			found = true
			break
		}
	}
	if !found {
		sd.serviceAddresses = append(sd.serviceAddresses, addr)
	}
}

// removeService 从服务地址列表中移除服务
func (sd *serviceDiscovery) removeService(addr string) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	for i, existingAddr := range sd.serviceAddresses {
		if existingAddr == addr {
			sd.serviceAddresses = append(sd.serviceAddresses[:i], sd.serviceAddresses[i+1:]...)
			break
		}
	}
}

// GetRandomServiceAddress 获取一个随机的服务地址 (简单的负载均衡)
func (sd *serviceDiscovery) GetRandomServiceAddress() (string, error) {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	if len(sd.serviceAddresses) == 0 {
		return "", errors.New("no available service addresses")
	}

	// 简单的随机负载均衡，确保每次运行的随机性
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	idx := r.Intn(len(sd.serviceAddresses))
	return sd.serviceAddresses[idx], nil
}

// --- Etcd 相关新增/修改 结束 ---

func main() {
	// --- Etcd 相关新增/修改 开始 ---
	// 初始化服务发现客户端
	sd, err := NewServiceDiscovery([]string{EtcdEndpoints}, EtcdServicePrefix)
	if err != nil {
		log.Fatalf("Failed to initialize service discovery: %v", err)
	}
	defer func() {
		fmt.Println("Closing service discovery client...")
		if err := sd.Close(); err != nil {
			log.Printf("Error closing service discovery client: %v", err)
		}
	}()
	// --- Etcd 相关新增/修改 结束 ---

	// 循环调用 RPC 服务
	for i := 0; i < 5; i++ { // 循环5次演示
		fmt.Printf("\n--- RPC Call Iteration %d ---\n", i+1)

		// --- Etcd 相关新增/修改 开始 ---
		// 1. 从服务发现中获取一个可用的服务器地址
		serverAddr, err := sd.GetRandomServiceAddress()
		if err != nil {
			fmt.Printf("No service available for RPC call: %v. Retrying in 2 seconds...\n", err)
			time.Sleep(2 * time.Second)
			continue // 继续下一次循环，尝试获取服务
		}
		fmt.Printf("Discovered RPC server address: %s\n", serverAddr)

		// 2. 连接 RPC 服务器 (使用 net.DialTimeout 建立连接，然后用 rpc.NewClient 包装)
		conn, err := net.DialTimeout("tcp", serverAddr, DialTimeout)
		if err != nil {
			fmt.Printf("Failed to dial TCP connection to %s: %v. Removing from service list and retrying in 2 seconds...\n", serverAddr, err)
			sd.removeService(serverAddr) // 连接失败，从本地服务列表中移除
			time.Sleep(2 * time.Second)
			continue // 继续下一次循环，尝试连接其他服务
		}

		client := rpc.NewClient(conn) // 将 net.Conn 包装成 rpc.Client

		// 确保 RPC client 关闭
		// 在循环内部的 defer 需要特别注意，每次迭代都会添加一个 defer
		// 更推荐的做法是将 RPC 调用逻辑封装到函数中，或在循环结束时手动关闭
		// 这里为了演示清晰，暂时保留 defer，但请注意其副作用
		defer func(c *rpc.Client) {
			if c != nil {
				if err := c.Close(); err != nil {
					log.Printf("Error closing RPC client: %v", err)
				}
			}
		}(client)

		// --- 3. 调用 CreateUser 方法 ---
		fmt.Println("--- Calling CreateUser ---")
		createReq := service.CreateUserRequest{
			Name:  fmt.Sprintf("User-%d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
			Age:   20 + i,
		}
		var createResp service.CreateUserResponse
		callErr := client.Call("UserService.CreateUser", createReq, &createResp)
		if callErr != nil {
			fmt.Printf("Failed to call CreateUser: %v\n", callErr)
			// 如果 RPC 调用失败，可能是服务不可用，可以尝试移除这个地址
			sd.removeService(serverAddr) // RPC 调用失败，从本地服务列表中移除
			continue                     // 继续下一次循环
		}
		if createResp.Error != "" {
			fmt.Printf("CreateUser business error: %s\n", createResp.Error)
		} else {
			fmt.Printf("User created: %+v\n", createResp.User)
		}
		createdUserID := createResp.User.ID

		// --- 4. 调用 GetUser 方法 ---
		fmt.Println("--- Calling GetUser (existing user) ---")
		getReq := service.GetUserRequest{ID: createdUserID}
		var getResp service.GetUserResponse
		callErr = client.Call("UserService.GetUser", getReq, &getResp)
		if callErr != nil {
			fmt.Printf("Failed to call GetUser: %v\n", callErr)
			sd.removeService(serverAddr) // RPC 调用失败，从本地服务列表中移除
			continue                     // 继续下一次循环
		}
		if getResp.Error != "" {
			fmt.Printf("GetUser business error: %s\n", getResp.Error)
		} else {
			fmt.Printf("User retrieved: %+v\n", getResp.User)
		}

		time.Sleep(time.Second) // 每次调用之间稍作间隔

		// 每次迭代结束后手动关闭 RPC 客户端连接，防止 defer 累积过多连接
		if client != nil {
			if err := client.Close(); err != nil {
				log.Printf("Error closing RPC client in loop: %v", err)
			}
		}
	}

	fmt.Println("\nAll RPC calls completed using service discovery.")
	time.Sleep(time.Second * 2) // 留点时间观察输出和Etcd的watch日志
}
