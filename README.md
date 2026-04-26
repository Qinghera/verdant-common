# Verdant Common

> **青禾 (QingHe)** 公共库  
> 所有后端服务共享的基础组件

---

## 简介

`verdant-common` 是青禾平台的公共库，继承自 [agamotto/go-common](https://github.com/agamotto-cloud/go-common) 并升级适配 Go 1.23+。

提供所有微服务需要的基础能力：
- HTTP服务器启动与优雅关闭
- Redis连接与缓存
- 配置中心 (本地YAML + Redis热更新)
- 服务发现 (Redis-based)
- 统一响应格式与错误码
- 日志与链路追踪
- gRPC客户端与解析器

---

## 安装

```bash
go get github.com/verdant-tech/verdant-common
```

---

## 模块列表

| 包 | 路径 | 说明 |
|----|------|------|
| `config` | `common/config` | 配置中心 |
| `redis` | `common/data/redis` | Redis客户端 |
| `db` | `common/data/db` | 数据库初始化 |
| `discovery` | `common/discovery` | 服务发现 |
| `logger` | `common/logger` | 日志与追踪 |
| `response` | `common/response` | 统一响应 |
| `param` | `common/param` | 通用参数结构 |
| `rpc` | `common/rpc` | gRPC客户端 |
| `start` | `common/start` | 服务启动 |

---

## 快速开始

### 启动HTTP服务

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/verdant-tech/verdant-common/common/response"
    "github.com/verdant-tech/verdant-common/common/start"
)

func main() {
    start.HttpServer(func(r *gin.Engine) {
        r.GET("/health", func(c *gin.Context) {
            response.OkWithData(gin.H{"status": "up"}, c)
        })
    })
}
```

### 读取配置

```go
import "github.com/verdant-tech/verdant-common/common/config"

// 自动从 config.yaml 和 Redis 加载配置
redisConfig := config.GetConfig("redis", redis.RedisConfig{})
serverConfig := config.GetServerConfig()
```

### 服务注册

```go
import "github.com/verdant-tech/verdant-common/common/discovery"

// 自动注册到Redis，每5秒心跳
discovery.SetGetServerInfoFunc(func(node discovery.ServerNode[any]) any {
    return gin.H{"version": "1.0.0"}
})
```

---

## 技术栈

- Go 1.23+
- Gin (HTTP框架)
- Redis (缓存、配置、服务发现)
- zerolog (结构化日志)
- gRPC + Protobuf (服务间通信)
- GORM (ORM)

---

## 项目结构

```
verdant-common/
├── common/
│   ├── config/          # 配置中心
│   ├── data/
│   │   ├── db/          # 数据库
│   │   └── redis/       # Redis
│   ├── discovery/       # 服务发现
│   ├── logger/          # 日志
│   ├── param/           # 参数
│   ├── response/        # 响应
│   ├── rpc/             # gRPC
│   └── start/           # 启动器
├── test/                # 测试模块
├── go.mod
├── go.work
└── README.md
```

---

## 继承与升级

本库继承自 [agamotto-cloud/go-common](https://github.com/agamotto-cloud/go-common) 的以下特性：
- ✅ Redis-based 服务发现
- ✅ 配置中心 (YAML + Redis)
- ✅ zerolog 日志
- ✅ 统一响应格式

升级改进：
- 🆙 Go 1.23+ 语法
- 🆙 泛型支持 (Go 1.18+)
- 🆙 更完善的错误包装
- 🆙 链路追踪增强

---

## 许可证

MIT License

---

> **青禾 (QingHe)** — 让技术如青禾般生长  
> [github.com/verdant-tech](https://github.com/verdant-tech)
