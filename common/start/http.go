package start

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/Qinghera/verdant-common/common/config"
	"github.com/Qinghera/verdant-common/common/discovery"
	"github.com/Qinghera/verdant-common/common/logger"
	"github.com/Qinghera/verdant-common/common/response"
)

// HttpServer 启动HTTP服务器
// routerReg: 路由注册回调函数，在此回调中注册业务路由
func HttpServer(routerReg func(r *gin.Engine)) {
	serverConfig := config.GetServerConfig()

	// 设置gin运行模式
	if serverConfig.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	log.Info().Int("port", serverConfig.Port).Str("name", serverConfig.Name).Msg("启动HTTP服务器")

	router := gin.New()

	// 全局中间件
	router.Use(gin.Recovery())
	router.Use(loggerHandle())
	router.Use(requestTimeout(30 * time.Second))

	// 404处理
	router.NoRoute(func(c *gin.Context) {
		response.NotFound(c)
	})

	// 用户路由注册
	if routerReg != nil {
		routerReg(router)
	}

	// 打印路由表 (debug模式)
	if gin.Mode() == gin.DebugMode {
		gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
			log.Debug().Str("method", httpMethod).Str("path", absolutePath).Str("handler", handlerName).Msg("注册路由")
		}
	}

	// 设置信任代理
	_ = router.SetTrustedProxies(nil)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:         ":" + strconv.Itoa(serverConfig.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动服务
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP服务器启动失败")
		}
	}()

	// 如果配置了网关连接，异步注册到网关
	if serverConfig.ConnectGateway != "" {
		go func() {
			time.Sleep(2 * time.Second) // 等待服务完全启动
			connectGateway(serverConfig.ConnectGateway, srv)
		}()
	}

	// 初始化服务发现
	discovery.Init()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("正在关闭服务器...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("服务器关闭失败")
	}

	log.Info().Msg("服务器已退出")
}

// loggerHandle 请求日志中间件
func loggerHandle() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 创建链路追踪Span
		ctx, span := logger.CreateSpan(c.Request.Context(), path)
		c.Request = c.Request.WithContext(ctx)
		defer span.End()

		// 记录请求开始
		log.Ctx(c.Request.Context()).Info().
			Str("method", c.Request.Method).
			Str("path", path).
			Str("ip", c.ClientIP()).
			Msg("请求开始")

		// 处理请求
		c.Next()

		// 记录请求结束
		latency := time.Since(start)
		status := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Ctx(c.Request.Context()).Info().
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", status).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Msg("请求完成")
	}
}

// requestTimeout 请求超时中间件
func requestTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		if ctx.Err() == context.DeadlineExceeded {
			response.ErrorWithCode(response.Timeout, ctx.Err(), c)
			c.Abort()
		}
	}
}
