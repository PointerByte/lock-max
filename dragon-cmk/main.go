// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"
	"net/http"

	serverGin "github.com/PointerByte/GoForge/config/server/gin"
	serverGRPC "github.com/PointerByte/GoForge/config/server/grpc"
	"github.com/PointerByte/GoForge/logger/builder"
	middlewaresSecurity "github.com/PointerByte/GoForge/security/middlewares"
	"github.com/PointerByte/GoForge/tools/workers"
	"github.com/PointerByte/lock-max/dragon-cmk/controller"
	"github.com/PointerByte/lock-max/dragon-cmk/entity/postgreSQL"
	pb "github.com/PointerByte/lock-max/dragon-cmk/proto"
	cmk "github.com/PointerByte/lock-max/dragon-cmk/service/CMK"
	kek "github.com/PointerByte/lock-max/dragon-cmk/service/KEK"
	auth "github.com/PointerByte/lock-max/dragon-cmk/service/auth"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

type grpcServer interface {
	Serve() error
	Register(register serverGRPC.RegisterServiceFunc) error
}

var (
	newGRPCServerFn = func() grpcServer {
		return serverGRPC.NewIConfig(nil, nil,
			serverGRPC.WithUnaryInterceptors(
				requireJWTWithAuthBypass(),
			),
		)
	}
	createAppFn = func() (*http.Server, error) {
		return serverGin.CreateApp()
	}
	newRepositoriesFn = func() (cmk.IRepository, kek.IRepository, auth.IRepository) {
		ctx := context.Background()
		return controller.NewRepositories(ctx, builder.New(ctx))
	}
	newGRPCServicesFn = func(keyRepository cmk.IRepository, authRepository auth.IRepository) pb.KeyServiceServer {
		return controller.NewKeyService(keyRepository, authRepository)
	}
	registerRESTRoutesFn  = controller.RegisterRESTRoutes
	startServerFn         = serverGin.Start
	setFunctionsRefreshFn = serverGin.SetFunctionsRefresh
	restartPoolFn         = postgreSQL.RestartPostgreSQLPool
	stopWorkersFn         = workers.StopWorkers
	runWorkersFn          = workers.RunWorkers
	loadEnvFilesFn        = func() {
		_ = godotenv.Load(".env", ".env.local")
	}
	panicFn = func(args ...any) {
		log.Panic(args...)
	}
)

var unauthenticatedGRPCMethods = map[string]struct{}{
	pb.KeyService_CreateServiceToken_FullMethodName: {},
	pb.KeyService_CreateAPIToken_FullMethodName:     {},
	pb.KeyService_CreateAPIClient_FullMethodName:    {},
	pb.KeyService_ListAPIClients_FullMethodName:     {},
	pb.KeyService_GetAPIClient_FullMethodName:       {},
	pb.KeyService_DeleteAPIClient_FullMethodName:    {},
}

func requireJWTWithAuthBypass() grpc.UnaryServerInterceptor {
	requireJWT := middlewaresSecurity.RequireJWTUnaryServerInterceptor()
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, ok := unauthenticatedGRPCMethods[info.FullMethod]; ok {
			return handler(ctx, req)
		}
		return requireJWT(ctx, req, info, handler)
	}
}

// @title Dragon CMK REST API
// @version 1.0
// @description REST API for Customer Managed Keys (CMK), whose purpose is to let customers control key lifecycle and protect data through managed encryption keys.
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Use a Bearer token in the format: Bearer {token}
func main() {
	loadEnvFilesFn()

	srv2, err := createAppFn()
	if err != nil {
		panicFn(err)
		return
	}

	restartWorker := func() error {
		stopWorkersFn()
		runWorkersFn()
		return nil
	}

	keyRepository, wrappingRepository, authRepository := newRepositoriesFn()
	registerRESTRoutesFn(keyRepository, wrappingRepository, authRepository)

	setFunctionsRefreshFn(restartPoolFn, restartWorker)
	go startServerFn(srv2)

	srv := newGRPCServerFn()
	keyService := newGRPCServicesFn(keyRepository, authRepository)
	if err := srv.Register(func(sr grpc.ServiceRegistrar) {
		pb.RegisterKeyServiceServer(sr, keyService)
	}); err != nil {
		panicFn(err)
		return
	}

	if err := srv.Serve(); err != nil {
		panicFn(err)
		return
	}
}
