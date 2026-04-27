package start

import (
	"fmt"
	"os"
)

// Start 启动服务
type Start struct {
	Name string
	Port string
}

// New 创建启动器
func New(name, port string) *Start {
	return &Start{
		Name: name,
		Port: port,
	}
}

// Run 运行服务
func (s *Start) Run() error {
	fmt.Printf("Starting %s on port %s\n", s.Name, s.Port)
	return nil
}

// Stop 停止服务
func (s *Start) Stop() error {
	fmt.Printf("Stopping %s\n", s.Name)
	return nil
}

// GetEnv 获取环境变量
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
