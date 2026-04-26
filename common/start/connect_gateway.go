package start

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/verdant-tech/verdant-common/common/config"
)

// connectGateway 连接到网关并注册服务
func connectGateway(gatewayURL string, srv *http.Server) {
	serverConfig := config.GetServerConfig()
	maxRetries := 5
	retryDelay := 3 * time.Second

	for i := 0; i < maxRetries; i++ {
		// 构建注册请求
		registerURL := fmt.Sprintf("%s/api/v1/admin/services/register", gatewayURL)

		// 发送注册请求
		resp, err := http.Post(registerURL, "application/json", nil)
		if err != nil {
			log.Error().
				Err(err).
				Str("gateway", gatewayURL).
				Int("retry", i+1).
				Msg("连接网关失败")
			time.Sleep(retryDelay)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Info().
				Str("gateway", gatewayURL).
				Str("service", serverConfig.Name).
				Int("port", serverConfig.Port).
				Msg("成功连接到网关")
			return
		}

		log.Warn().
			Str("gateway", gatewayURL).
			Int("status", resp.StatusCode).
			Int("retry", i+1).
			Msg("网关注册返回非200状态码")
		time.Sleep(retryDelay)
	}

	log.Error().
		Str("gateway", gatewayURL).
		Int("retries", maxRetries).
		Msg("连接网关失败，已达到最大重试次数")
}
