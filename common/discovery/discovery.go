package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/Qinghera/verdant-common/common/config"
	verdantRedis "github.com/Qinghera/verdant-common/common/data/redis"
)

// ServerNode 服务节点信息
type ServerNode[T any] struct {
	ID             string `json:"id"`             // 实例ID (唯一)
	ServiceName    string `json:"serviceName"`    // 服务名
	ActiveLastTime int64  `json:"activeLastTime"` // 最后心跳时间
	Address        string `json:"address"`        // IP地址
	Port           int    `json:"port"`           // 端口
	Weight         int    `json:"weight"`         // 权重
	Status         int    `json:"status"`         // 状态: 0离线 1在线 2维护中
	Info           T      `json:"info"`           // 扩展信息
}

// Registry 服务注册接口
type Registry interface {
	Register(node ServerNode[any]) error
	Deregister(serviceName, instanceID string) error
	Discover(serviceName string) ([]ServerNode[any], error)
	Heartbeat(serviceName, instanceID string) error
}

// redisRegistry Redis注册中心实现
type redisRegistry struct {
	client *redis.Client
}

var (
	// registryInstance 全局注册中心实例
	registryInstance Registry
	registryOnce     sync.Once

	// serverNodeInfo 当前服务节点信息
	serverNodeInfo = ServerNode[any]{}

	// getServerInfoFunc 用户自定义信息回调
	getServerInfoFunc func(serverNode ServerNode[any]) any

	// instanceID 当前实例ID
	instanceID string
)

// GetRegistry 获取注册中心实例
func GetRegistry() Registry {
	registryOnce.Do(func() {
		registryInstance = &redisRegistry{
			client: verdantRedis.RedisClient,
		}
	})
	return registryInstance
}

// Init 初始化服务发现
// 自动注册到Redis，启动心跳协程
func Init() {
	serverConfig := config.GetServerConfig()

	serverNodeInfo = ServerNode[any]{
		ID:             generateInstanceID(serverConfig.Name, serverConfig.Port),
		ServiceName:    serverConfig.Name,
		ActiveLastTime: time.Now().Unix(),
		Address:        getLocalIP(),
		Port:           serverConfig.Port,
		Weight:         1,
		Status:         1,
	}

	instanceID = serverNodeInfo.ID

	// 注册服务
	if err := GetRegistry().Register(serverNodeInfo); err != nil {
		log.Error().Err(err).Msg("服务注册失败")
	}

	// 启动心跳协程
	go heartbeatLoop()

	// 监听退出信号
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Info().Msg("接收到退出信号，注销服务")
		if err := GetRegistry().Deregister(serverConfig.Name, instanceID); err != nil {
			log.Error().Err(err).Msg("服务注销失败")
		}
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()

	log.Info().
		Str("id", instanceID).
		Str("service", serverConfig.Name).
		Str("address", serverNodeInfo.Address).
		Int("port", serverConfig.Port).
		Msg("服务注册成功")
}

// SetServerInfoFunc 设置服务信息回调函数
func SetServerInfoFunc(f func(serverNode ServerNode[any]) any) {
	getServerInfoFunc = f
}

// GetServiceList 获取服务列表
func GetServiceList[T any](serviceName string) []ServerNode[T] {
	ctx := context.Background()
	serverKey := fmt.Sprintf("service:%s", serviceName)

	result, err := verdantRedis.RedisClient.HGetAll(ctx, serverKey).Result()
	if err != nil {
		log.Error().Err(err).Str("service", serviceName).Msg("获取服务列表失败")
		return nil
	}

	return mapsToServerList[T](result)
}

// DiscoverHealthy 发现健康的服务实例
func DiscoverHealthy(serviceName string) ([]ServerNode[any], error) {
	all := GetServiceList[any](serviceName)
	if all == nil {
		return nil, fmt.Errorf("服务 %s 不存在", serviceName)
	}

	var healthy []ServerNode[any]
	now := time.Now().Unix()
	for _, node := range all {
		// 检查心跳超时 (10分钟)
		if now-node.ActiveLastTime > 600 {
			continue
		}
		// 检查状态
		if node.Status != 1 {
			continue
		}
		healthy = append(healthy, node)
	}

	if len(healthy) == 0 {
		return nil, fmt.Errorf("服务 %s 无健康实例", serviceName)
	}

	return healthy, nil
}

// Register 手动注册服务
func (r *redisRegistry) Register(node ServerNode[any]) error {
	ctx := context.Background()
	serverKey := fmt.Sprintf("service:%s", node.ServiceName)
	field := fmt.Sprintf("%s:%d", node.Address, node.Port)

	jsonStr, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("序列化服务节点失败: %w", err)
	}

	if err := r.client.HSet(ctx, serverKey, field, jsonStr).Err(); err != nil {
		return fmt.Errorf("Redis注册失败: %w", err)
	}

	// 设置TTL
	if err := r.client.Expire(ctx, serverKey, time.Hour).Err(); err != nil {
		log.Warn().Err(err).Msg("设置Redis TTL失败")
	}

	return nil
}

// Deregister 注销服务
func (r *redisRegistry) Deregister(serviceName, instanceID string) error {
	ctx := context.Background()
	serverKey := fmt.Sprintf("service:%s", serviceName)

	// 获取所有实例，找到匹配的
	result, err := r.client.HGetAll(ctx, serverKey).Result()
	if err != nil {
		return err
	}

	for field, val := range result {
		var node ServerNode[any]
		if err := json.Unmarshal([]byte(val), &node); err != nil {
			continue
		}
		if node.ID == instanceID {
			return r.client.HDel(ctx, serverKey, field).Err()
		}
	}

	return fmt.Errorf("实例 %s 不存在", instanceID)
}

// Discover 发现服务
func (r *redisRegistry) Discover(serviceName string) ([]ServerNode[any], error) {
	return DiscoverHealthy(serviceName)
}

// Heartbeat 发送心跳
func (r *redisRegistry) Heartbeat(serviceName, instanceID string) error {
	ctx := context.Background()
	serverKey := fmt.Sprintf("service:%s", serviceName)

	result, err := r.client.HGetAll(ctx, serverKey).Result()
	if err != nil {
		return err
	}

	for field, val := range result {
		var node ServerNode[any]
		if err := json.Unmarshal([]byte(val), &node); err != nil {
			continue
		}
		if node.ID == instanceID {
			node.ActiveLastTime = time.Now().Unix()
			jsonStr, _ := json.Marshal(node)
			return r.client.HSet(ctx, serverKey, field, jsonStr).Err()
		}
	}

	return fmt.Errorf("实例 %s 不存在", instanceID)
}

// heartbeatLoop 心跳循环
func heartbeatLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		serverConfig := config.GetServerConfig()
		serverNodeInfo.ActiveLastTime = time.Now().Unix()

		// 调用用户自定义回调
		if getServerInfoFunc != nil {
			serverNodeInfo.Info = getServerInfoFunc(serverNodeInfo)
		}

		// 更新注册信息
		if err := GetRegistry().Heartbeat(serverConfig.Name, instanceID); err != nil {
			// 如果心跳失败，重新注册
			log.Warn().Err(err).Msg("心跳失败，尝试重新注册")
			if err := GetRegistry().Register(serverNodeInfo); err != nil {
				log.Error().Err(err).Msg("重新注册失败")
			}
		}

		// 清理超时实例
		cleanupExpiredInstances(serverConfig.Name)
	}
}

// cleanupExpiredInstances 清理超时实例
func cleanupExpiredInstances(serviceName string) {
	ctx := context.Background()
	serverKey := fmt.Sprintf("service:%s", serviceName)

	result, err := verdantRedis.RedisClient.HGetAll(ctx, serverKey).Result()
	if err != nil {
		return
	}

	now := time.Now().Unix()
	for field, val := range result {
		var node ServerNode[any]
		if err := json.Unmarshal([]byte(val), &node); err != nil {
			continue
		}
		// 超过10分钟无心跳，删除
		if now-node.ActiveLastTime > 600 {
			verdantRedis.RedisClient.HDel(ctx, serverKey, field)
			log.Info().Str("instance", node.ID).Msg("清理超时实例")
		}
	}
}

// mapsToServerList 将Redis Hash转换为服务列表
func mapsToServerList[T any](result map[string]string) []ServerNode[T] {
	var serverList = make([]ServerNode[T], 0)
	for _, v := range result {
		var serverNode ServerNode[T]
		err := json.Unmarshal([]byte(v), &serverNode)
		if err != nil {
			log.Error().Err(err).Msg("解析服务节点失败")
			continue
		}
		serverList = append(serverList, serverNode)
	}
	return serverList
}

// generateInstanceID 生成实例ID
func generateInstanceID(serviceName string, port int) string {
	ip := getLocalIP()
	return fmt.Sprintf("%s-%s-%d", serviceName, ip, port)
}

// getLocalIP 获取本机IP
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, address := range addrs {
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}
