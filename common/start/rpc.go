package start

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"github.com/rs/zerolog/log"
	"github.com/Qinghera/verdant-common/common/config"
)

// RpcServer 启动gRPC服务器
// register: 服务注册回调函数，在此回调中注册gRPC服务
func RpcServer(register func(s *grpc.Server)) {
	serverConfig := config.GetServerConfig()
	port := serverConfig.Port

	log.Info().Int("port", port).Str("name", serverConfig.Name).Msg("启动gRPC服务器")

	// 创建gRPC服务器
	s := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
		grpc.StreamInterceptor(streamInterceptor),
	)

	// 注册服务
	if register != nil {
		register(s)
	}

	// 监听端口
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal().Err(err).Int("port", port).Msg("gRPC监听失败")
	}

	// 启动服务
	if err := s.Serve(lis); err != nil {
		log.Fatal().Err(err).Msg("gRPC服务器启动失败")
	}
}

// unaryInterceptor 一元调用拦截器
func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Info().Str("method", info.FullMethod).Msg("gRPC请求")
	resp, err := handler(ctx, req)
	if err != nil {
		log.Error().Err(err).Str("method", info.FullMethod).Msg("gRPC请求失败")
	}
	return resp, err
}

// streamInterceptor 流式调用拦截器
func streamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Info().Str("method", info.FullMethod).Msg("gRPC流式请求")
	err := handler(srv, ss)
	if err != nil {
		log.Error().Err(err).Str("method", info.FullMethod).Msg("gRPC流式请求失败")
	}
	return err
}
