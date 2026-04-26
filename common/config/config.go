package config

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"github.com/verdant-tech/verdant-common/common/data/redis"
)

// ConfigProps 全部配置 (合并后的)
var ConfigProps map[string]interface{}

// ConfigMap 格式化后的配置缓存
var ConfigMap = make(map[string]interface{})

// configMutex 配置读写锁
var configMutex sync.RWMutex

// watchers 配置监听器
var watchers = make(map[string][]func(oldVal, newVal interface{}))
var watcherMutex sync.RWMutex

// env 当前运行环境
var env string

func init() {
	name := flag.String("env", "dev", "运行环境 (dev/staging/prod)")
	flag.Parse()
	env = *name
	if env == "" {
		env = "dev"
	}

	log.Info().Str("env", env).Msg("加载服务器配置")

	// 1. 读取本地配置文件
	configData, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatal().Msgf("加载配置文件失败: %v", err)
	}

	err = yaml.Unmarshal(configData, &ConfigProps)
	if err != nil {
		log.Fatal().Msgf("读取配置文件失败: %v", err)
	}

	serverName := ConfigProps["server"].(map[string]interface{})["name"].(string)

	// 2. 初始化 Redis
	redisConfig := GetConfig("redis", redis.RedisConfig{})
	redis.InitRedis(redisConfig)

	// 3. 从 Redis 获取配置并合并
	configKey := fmt.Sprintf("config:service:%s:%s", serverName, env)
	redisConfigData, err := redis.RedisClient.Get(context.Background(), configKey).Result()
	if err != nil && err != redis.Nil {
		log.Warn().Err(err).Str("key", configKey).Msg("读取Redis配置失败")
	}

	if err == nil && redisConfigData != "" {
		var redisMap map[string]interface{}
		err = yaml.Unmarshal([]byte(redisConfigData), &redisMap)
		if err != nil {
			log.Error().Err(err).Msg("解析Redis配置失败")
		} else {
			// 合并配置 (Redis配置优先)
			for k, v := range redisMap {
				oldVal := ConfigProps[k]
				ConfigProps[k] = v
				// 触发监听器
				triggerWatchers(k, oldVal, v)
			}
			log.Info().Int("count", len(redisMap)).Msg("从Redis加载配置")
		}
	}

	// 4. 启动配置热更新监听
	go watchRemoteConfig(serverName)
}

// GetServerConfig 获取服务配置
func GetServerConfig() *ServerConfig {
	return GetConfig("server", ServerConfig{})
}

// GetConfig 获取指定配置项
// T: 目标类型
// configKey: 配置键名 (如 "redis", "database")
// defaultVal: 默认值
func GetConfig[T any](configKey string, defaultVal T) *T {
	configMutex.RLock()
	defer configMutex.RUnlock()

	// 如果之前获取过配置，直接返回缓存
	if cached, ok := ConfigMap[configKey]; ok {
		if typed, ok := cached.(T); ok {
			return &typed
		}
	}

	// 从 ConfigProps 解析
	val := ConfigProps[configKey]
	if val == nil {
		log.Warn().Str("key", configKey).Msg("配置项不存在，使用默认值")
		ConfigMap[configKey] = defaultVal
		return &defaultVal
	}

	var result T
	err := mapstructure.Decode(val, &result)
	if err != nil {
		log.Error().Err(err).Str("key", configKey).Msg("配置解析失败，使用默认值")
		ConfigMap[configKey] = defaultVal
		return &defaultVal
	}

	ConfigMap[configKey] = result
	return &result
}

// GetString 获取字符串配置
func GetString(key string) string {
	configMutex.RLock()
	defer configMutex.RUnlock()

	val := ConfigProps[key]
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}

// GetInt 获取整数配置
func GetInt(key string) int {
	configMutex.RLock()
	defer configMutex.RUnlock()

	val := ConfigProps[key]
	if val == nil {
		return 0
	}
	if i, ok := val.(int); ok {
		return i
	}
	if f, ok := val.(float64); ok {
		return int(f)
	}
	return 0
}

// GetBool 获取布尔配置
func GetBool(key string) bool {
	configMutex.RLock()
	defer configMutex.RUnlock()

	val := ConfigProps[key]
	if val == nil {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

// Watch 监听配置变化
func Watch(key string, callback func(oldVal, newVal interface{})) {
	watcherMutex.Lock()
	defer watcherMutex.Unlock()

	watchers[key] = append(watchers[key], callback)
	log.Info().Str("key", key).Msg("注册配置监听器")
}

// Reload 重新加载配置
func Reload() error {
	configMutex.Lock()
	defer configMutex.Unlock()

	// 清空缓存
	ConfigMap = make(map[string]interface{})

	// 重新读取本地配置
	configData, err := os.ReadFile("config.yaml")
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %w", err)
	}

	var newProps map[string]interface{}
	err = yaml.Unmarshal(configData, &newProps)
	if err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 对比变化并触发监听器
	for k, newVal := range newProps {
		oldVal := ConfigProps[k]
		if fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal) {
			triggerWatchers(k, oldVal, newVal)
		}
	}

	ConfigProps = newProps
	log.Info().Msg("配置已重新加载")
	return nil
}

// triggerWatchers 触发配置监听器
func triggerWatchers(key string, oldVal, newVal interface{}) {
	watcherMutex.RLock()
	defer watcherMutex.RUnlock()

	if cbs, ok := watchers[key]; ok {
		for _, cb := range cbs {
			go cb(oldVal, newVal)
		}
	}
}

// watchRemoteConfig 监听远程配置变化
func watchRemoteConfig(serverName string) {
	configKey := fmt.Sprintf("config:service:%s:%s", serverName, env)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		redisConfigData, err := redis.RedisClient.Get(context.Background(), configKey).Result()
		if err != nil {
			continue
		}

		var redisMap map[string]interface{}
		err = yaml.Unmarshal([]byte(redisConfigData), &redisMap)
		if err != nil {
			continue
		}

		configMutex.Lock()
		for k, newVal := range redisMap {
			oldVal := ConfigProps[k]
			if fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal) {
				ConfigProps[k] = newVal
				// 清空缓存
				delete(ConfigMap, k)
				configMutex.Unlock()
				triggerWatchers(k, oldVal, newVal)
				configMutex.Lock()
			}
		}
		configMutex.Unlock()
	}
}

// GetEnv 获取当前运行环境
func GetEnv() string {
	return env
}
