package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RedisClient 全局Redis客户端
var RedisClient *redis.Client

// RedisConfig Redis配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool-size"`
}

// InitRedis 初始化Redis连接
func InitRedis(redisConfig *RedisConfig) {
	if redisConfig == nil {
		redisConfig = &RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
			PoolSize: 10,
		}
	}

	log.Info().Str("addr", redisConfig.Addr).Int("db", redisConfig.DB).Msg("连接Redis")

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisConfig.Addr,
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
		PoolSize: redisConfig.PoolSize,
	})

	RedisClient = rdb

	// 异步检查连接
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := RedisClient.Ping(ctx).Result()
		if err != nil {
			log.Panic().Err(err).Msg("Redis连接失败")
		}

		info := RedisClient.Info(ctx, "server")
		versionStr := info.String()
		versionIndex := strings.Index(versionStr, "redis_version:")
		if versionIndex >= 0 {
			version := versionStr[versionIndex+14 : versionIndex+20]
			log.Info().Str("version", version).Msg("Redis连接成功")
		}
	}()
}

// Close 关闭Redis连接
func Close() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}

// Health 检查Redis健康状态
func Health(ctx context.Context) error {
	if RedisClient == nil {
		return fmt.Errorf("Redis未初始化")
	}
	_, err := RedisClient.Ping(ctx).Result()
	return err
}
