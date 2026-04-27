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
