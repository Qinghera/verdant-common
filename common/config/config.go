package config

import (
	"os"
	"strconv"
)

// Config 全局配置
type Config struct {
	Env       string
	Port      string
	DBPath    string
	JWTSecret string
	JWTExpire int
	Mysql     MysqlConfig
}

// MysqlConfig MySQL配置
type MysqlConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Charset  string
}

// ServerConfig 服务配置
type ServerConfig struct {
	Port           int    `mapstructure:"port"`
	Name           string `mapstructure:"name"`
	ConnectGateway string `mapstructure:"connect-gateway"`
	Mode           string `mapstructure:"mode"` // debug/release
}

// PostgresConfig PostgreSQL配置
type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"sslmode"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool-size"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret        string `mapstructure:"secret"`
	Expire        int    `mapstructure:"expire"`         // 访问令牌过期时间(秒)
	RefreshExpire int    `mapstructure:"refresh-expire"` // 刷新令牌过期时间(秒)
	Issuer        string `mapstructure:"issuer"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`       // debug/info/warn/error
	Format     string `mapstructure:"format"`      // json/console
	Output     string `mapstructure:"output"`      // stdout/file
	FilePath   string `mapstructure:"file-path"`   // 日志文件路径
	MaxSize    int    `mapstructure:"max-size"`    // 单个文件最大大小(MB)
	MaxBackups int    `mapstructure:"max-backups"` // 保留旧文件个数
	MaxAge     int    `mapstructure:"max-age"`     // 保留天数
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled   bool `mapstructure:"enabled"`
	Requests  int  `mapstructure:"requests"`  // 窗口内请求数
	WindowSec int  `mapstructure:"window-sec"` // 窗口大小(秒)
}

// CorsConfig CORS配置
type CorsConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed-origins"`
	AllowedMethods   []string `mapstructure:"allowed-methods"`
	AllowedHeaders   []string `mapstructure:"allowed-headers"`
	AllowCredentials bool     `mapstructure:"allow-credentials"`
	MaxAge           int      `mapstructure:"max-age"`
}

// WechatConfig 微信配置
type WechatConfig struct {
	AppID          string `mapstructure:"app-id"`
	AppSecret      string `mapstructure:"app-secret"`
	Token          string `mapstructure:"token"`
	EncodingAESKey string `mapstructure:"encoding-aes-key"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Provider  string `mapstructure:"provider"` // oss/cos/s3/r2
	Endpoint  string `mapstructure:"endpoint"`
	Bucket    string `mapstructure:"bucket"`
	Region    string `mapstructure:"region"`
	AccessKey string `mapstructure:"access-key"`
	SecretKey string `mapstructure:"secret-key"`
}

// GetConfig 获取配置
func GetConfig() *Config {
	return &Config{
		Env:       getEnv("ENV", "development"),
		Port:      getEnv("PORT", "8082"),
		DBPath:    getEnv("DB_PATH", "./data/admin.db"),
		JWTSecret: getEnv("JWT_SECRET", "qinghe-secret"),
		JWTExpire: getEnvInt("JWT_EXPIRE", 24),
		Mysql: MysqlConfig{
			Host:     getEnv("MYSQL_HOST", "localhost"),
			Port:     getEnvInt("MYSQL_PORT", 3306),
			User:     getEnv("MYSQL_USER", "root"),
			Password: getEnv("MYSQL_PASSWORD", ""),
			Database: getEnv("MYSQL_DATABASE", "verdant"),
			Charset:  getEnv("MYSQL_CHARSET", "utf8mb4"),
		},
	}
}

// GetServerConfig 获取服务配置
func GetServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:           getEnvInt("PORT", 8080),
		Name:           getEnv("SERVICE_NAME", "gateway"),
		ConnectGateway: getEnv("CONNECT_GATEWAY", ""),
		Mode:           getEnv("MODE", "release"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
