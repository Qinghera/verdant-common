# Verdant Common - 详细设计文档

> **版本**: 1.0  
> **日期**: 2026-04-24  
> **模块**: verdant-common (公共库)  
> **设计模式**: 结构 → 代码结构 → 表结构 → 接口 → 页面 → 功能

---

## 1. 结构 (Structure)

### 1.1 模块划分

```
verdant-common/
├── common/
│   ├── config/              # 配置中心
│   │   ├── config.go        # 配置加载与合并
│   │   ├── configStruct.go  # 配置结构体定义
│   │   └── watcher.go       # 配置热更新监听 (新增)
│   │
│   ├── data/
│   │   ├── db/              # 数据库初始化
│   │   │   └── dataInit.go  # GORM初始化
│   │   └── redis/           # Redis客户端
│   │       └── redis.go     # go-redis封装
│   │
│   ├── discovery/           # 服务发现
│   │   ├── discovery.go     # 服务注册与心跳
│   │   ├── get_service_list.go  # 服务列表查询
│   │   └── resolver.go      # gRPC服务解析 (新增)
│   │
│   ├── logger/              # 日志与链路追踪
│   │   ├── tracer.go        # OpenTelemetry追踪
│   │   └── zap_log.go       # zerolog配置
│   │
│   ├── param/               # 通用参数结构
│   │   ├── menu_param.go    # 菜单参数
│   │   ├── menu_sort_param.go
│   │   └── user_param.go    # 用户参数
│   │
│   ├── response/            # 统一响应格式
│   │   ├── ErrorCodes.go    # 错误码定义
│   │   └── common.go        # 响应工具函数
│   │
│   ├── rpc/                 # gRPC客户端
│   │   ├── client.go        # gRPC连接管理
│   │   └── resolver.go      # 自定义解析器
│   │
│   └── start/               # 服务启动器
│       ├── http.go          # HTTP服务器启动
│       ├── rpc.go           # gRPC服务器启动
│       └── connect_gateway.go  # 网关连接注册
│
├── test/                    # 测试模块
│   ├── main.go              # 测试入口
│   └── rpc/                 # RPC测试
│
├── go.mod
├── go.work
└── README.md
```

### 1.2 包依赖关系

```
config
  ├── redis (初始化Redis连接)
  └── yaml/mapstructure (解析配置)

discovery
  ├── config (获取服务名和端口)
  └── redis (服务注册到Redis)

start
  ├── config (获取端口配置)
  ├── logger (链路追踪)
  ├── response (404处理)
  └── discovery (连接网关时注册)

rpc
  └── discovery (服务发现解析器)
```

---

## 2. 代码结构 (Code Structure)

### 2.1 Go版本与依赖

```go
// go.mod
module github.com/verdant-tech/verdant-common

go 1.23

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/redis/go-redis/v9 v9.3.0
    github.com/rs/zerolog v1.31.0
    github.com/mitchellh/mapstructure v1.5.0
    gopkg.in/yaml.v3 v3.0.1
    gorm.io/gorm v1.25.5
    gorm.io/driver/mysql v1.5.2
    google.golang.org/grpc v1.59.0
    go.opentelemetry.io/otel v1.21.0
)
```

### 2.2 核心接口定义

```go
// common/config/config.go

package config

// ConfigLoader 配置加载器接口
type ConfigLoader interface {
    Load(path string) error
    Get(key string) interface{}
    GetString(key string) string
    GetInt(key string) int
    GetBool(key string) bool
    Watch(key string, callback func(oldVal, newVal interface{}))
}

// Provider 配置提供者接口 (支持多种来源)
type Provider interface {
    Load() (map[string]interface{}, error)
    Watch(callback func(changedKeys []string))
}
```

```go
// common/discovery/discovery.go

package discovery

// Registry 服务注册接口
type Registry interface {
    Register(node ServerNode) error
    Deregister(serviceName, instanceID string) error
    Discover(serviceName string) ([]ServerNode, error)
    Heartbeat(serviceName, instanceID string) error
}

// ServerNode 服务节点信息
type ServerNode[T any] struct {
    ID             string `json:"id"`             // 实例ID (唯一)
    ServiceName    string `json:"serviceName"`    // 服务名
    ActiveLastTime int64  `json:"activeLastTime"` // 最后心跳时间
    Address        string `json:"address"`        // IP地址
    Port           int    `json:"port"`           // 端口
    Weight         int    `json:"weight"`         // 权重 (新增)
    Status         int    `json:"status"`         // 状态: 0离线 1在线 2维护中 (新增)
    Info           T      `json:"info"`           // 扩展信息
}
```

```go
// common/response/response.go

package response

// Responder 响应器接口
type Responder interface {
    Ok(data interface{})
    OkWithMessage(message string)
    OkWithData(data interface{})
    Error(err error)
    Fail(message string)
    Result(code ErrorCode, data interface{}, message string)
}

// ErrorCode 错误码
type ErrorCode int

const (
    Success       ErrorCode = 0
    ParamError    ErrorCode = 1
    NetworkError  ErrorCode = 2
    NotFoundError ErrorCode = 404
    SystemError   ErrorCode = 7
    Unauthorized  ErrorCode = 401
    Forbidden     ErrorCode = 403
    IdNotEmpty    ErrorCode = 50001
)
```

---

## 3. 表结构 (Database Schema)

> verdant-common 是纯工具库，无业务表结构。
> Redis中存储的数据结构如下：

### 3.1 Redis数据结构

```
# 服务注册表 (Hash)
Key: service:{serviceName}
Field: {address}:{port}
Value: JSON(ServerNode)
TTL: 1小时 (每次心跳刷新)

示例:
HGETALL service:gateway
1) "192.168.1.100:8080"
   '{"id":"gateway-1","serviceName":"gateway","activeLastTime":1713980000,"address":"192.168.1.100","port":8080,"weight":1,"status":1}'
2) "192.168.1.101:8080"
   '{"id":"gateway-2","serviceName":"gateway","activeLastTime":1713980000,"address":"192.168.1.101","port":8080,"weight":1,"status":1}'

# 配置存储 (String)
Key: config:service:{serviceName}:{env}
Value: YAML格式的配置内容

示例:
GET config:service:gateway:dev
'
server:
  port: 8080
redis:
  addr: localhost:6379
'

# 限流计数器 (String + Expire)
Key: rate_limit:{client_id}:{path}
Value: 当前计数
Expire: 1分钟 (滑动窗口)
```

---

## 4. 接口 (API Design)

> verdant-common 是库，不提供HTTP接口。
> 以下为公共API（Go函数接口）：

### 4.1 config包 API

```go
// 初始化配置（自动调用）
// 从 config.yaml 加载本地配置
// 从 Redis 加载远程配置并合并
func init()

// 获取服务配置
func GetServerConfig() *ServerConfig

// 获取指定配置项
// T: 目标类型
// configKey: 配置键名 (如 "redis", "database")
// defaultVal: 默认值
func GetConfig[T any](configKey string, defaultVal T) *T

// 获取字符串配置
func GetString(key string) string

// 获取整数配置
func GetInt(key string) int

// 监听配置变化 (新增)
func Watch(key string, callback func(oldVal, newVal interface{}))

// 重新加载配置 (新增)
func Reload() error
```

### 4.2 discovery包 API

```go
// 设置服务信息回调函数
// 用于在心跳时上报自定义信息
func SetServerInfoFunc[T any](f func(serverNode ServerNode[T]) T)

// 获取服务列表
func GetServiceList[T any](serviceName string) []ServerNode[T]

// 手动注册服务 (新增)
func Register(node ServerNode) error

// 手动注销服务 (新增)
func Deregister(serviceName, instanceID string) error

// 发现健康的服务实例 (新增)
func DiscoverHealthy(serviceName string) ([]ServerNode, error)
```

### 4.3 response包 API

```go
// 成功响应 (无数据)
func Ok(c *gin.Context)

// 成功响应 (带消息)
func OkWithMessage(message string, c *gin.Context)

// 成功响应 (带数据)
func OkWithData(data interface{}, c *gin.Context)

// 成功响应 (带数据和消息)
func OkWithDetailed(data interface{}, message string, c *gin.Context)

// 错误响应
func Error(err error, c *gin.Context)

// 失败响应 (系统错误)
func FailWithMessage(message string, c *gin.Context)

// 失败响应 (带数据)
func FailWithDetailed(data interface{}, message string, c *gin.Context)

// 通用响应
func Result(code ErrorCode, data interface{}, msg string, c *gin.Context)
```

### 4.4 start包 API

```go
// 启动HTTP服务器
// routerReg: 路由注册回调函数
func HttpServer(routerReg func(r *gin.Engine))

// 启动gRPC服务器 (新增)
// register: 服务注册回调函数
func RpcServer(register func(s *grpc.Server))
```

### 4.5 redis包 API

```go
// 初始化Redis连接
func InitRedis(redisConfig *RedisConfig)

// 获取Redis客户端实例
var RedisClient *redis.Client

// RedisConfig Redis配置
type RedisConfig struct {
    Addr     string
    Password string
    DB       int      // 新增: 数据库编号
    PoolSize int      // 新增: 连接池大小
}
```

---

## 5. 页面 (Page Design)

> verdant-common 是纯后端库，无前端页面。

---

## 6. 功能 (Feature Design)

### 6.1 配置加载流程

```
1. 读取命令行参数 -env (默认 dev)
2. 读取本地 config.yaml
3. 解析 server.name 确定服务名
4. 从 config.yaml 读取 redis 配置
5. 初始化 Redis 连接
6. 从 Redis 读取远程配置 (config:service:{name}:{env})
7. 合并本地配置和远程配置 (远程优先)
8. 缓存到 ConfigMap 避免重复解析
```

### 6.2 服务发现流程

```
启动时:
1. 获取本机IP (非回环地址)
2. 获取服务端口号 (从配置)
3. 生成实例ID ({serviceName}-{ip}-{port})
4. 注册到Redis (service:{serviceName} Hash)
5. 启动心跳协程 (每5秒)

运行时:
1. 每5秒更新心跳时间
2. 调用用户自定义的 info 函数获取扩展信息
3. 写入Redis并刷新TTL
4. 清理超时实例 (超过10分钟无心跳)

关闭时:
1. 捕获 SIGINT/SIGTERM
2. 从Redis注销实例
3. 延迟1秒后退出
```

### 6.3 HTTP服务器启动流程

```
1. 读取 server.port 配置
2. 创建 gin.Engine
3. 注册全局中间件:
   - Recovery ( panic 恢复)
   - Logger (请求日志 + 链路追踪)
   - NoRoute (404处理)
4. 调用用户路由注册回调
5. 打印路由表 (debug模式)
6. 启动 http.Server
7. 如果配置了 connectGateway，异步连接网关
8. 等待退出信号 (SIGINT)
9. 优雅关闭 (5秒超时)
```

### 6.4 链路追踪流程

```
每个请求:
1. 创建 Span (OpenTelemetry)
2. 注入 TraceID 到 Context
3. 记录请求开始日志
4. 执行 Handler
5. 记录请求结束日志 (状态码、耗时)
6. 结束 Span
```

---

## 7. 升级改进 (相比 agamotto/go-common)

| 改进项 | 原代码 | 升级后 |
|--------|--------|--------|
| Go版本 | 1.20 | 1.23 |
| 泛型支持 | 部分使用 | 全面使用 |
| 配置监听 | 无 | Watch() 热更新 |
| 服务权重 | 无 | Weight 字段 |
| 服务状态 | 无 | Status 字段 |
| Redis DB | 固定0 | 可配置 |
| 连接池 | 默认 | 可配置 PoolSize |
| 优雅关闭 | 5秒固定 | 可配置超时 |
| 错误码 | 基础 | 增加 Unauthorized/Forbidden |
| 配置重载 | 无 | Reload() 方法 |

---

## 8. 文件清单

```
verdant-common/
├── DESIGN.md                    # 本文件
├── STRUCTURE.md                 # 结构说明
├── API.md                       # 接口文档
├── DATABASE.md                  # Redis数据结构
├── go.mod
├── go.work
├── README.md
├── common/
│   ├── config/
│   │   ├── config.go            # 配置加载
│   │   ├── configStruct.go      # 结构体定义
│   │   └── watcher.go           # 热更新监听 [新增]
│   ├── data/
│   │   ├── db/
│   │   │   └── dataInit.go      # 数据库初始化
│   │   └── redis/
│   │       └── redis.go         # Redis客户端
│   ├── discovery/
│   │   ├── discovery.go         # 服务发现
│   │   ├── get_service_list.go  # 服务列表
│   │   └── resolver.go          # gRPC解析器 [新增]
│   ├── logger/
│   │   ├── tracer.go            # 链路追踪
│   │   └── zap_log.go           # 日志配置
│   ├── param/
│   │   ├── menu_param.go
│   │   ├── menu_sort_param.go
│   │   └── user_param.go
│   ├── response/
│   │   ├── ErrorCodes.go        # 错误码
│   │   └── common.go            # 响应工具
│   ├── rpc/
│   │   ├── client.go            # gRPC客户端
│   │   └── resolver.go          # 服务解析器
│   └── start/
│       ├── http.go              # HTTP启动器
│       ├── rpc.go               # gRPC启动器 [新增]
│       └── connect_gateway.go   # 网关连接
└── test/
    ├── main.go
    └── rpc/
        ├── rpc.go
        └── user.go
```

---

> **设计者**: Hermes Agent  
> **日期**: 2026-04-24  
> **状态**: 已完成设计
