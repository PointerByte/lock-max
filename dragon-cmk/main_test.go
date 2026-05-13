// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"net/http"
	"sort"
	"testing"
	"time"

	serverGin "github.com/PointerByte/GoForge/config/server/gin"
	serverGRPC "github.com/PointerByte/GoForge/config/server/grpc"
	pb "github.com/PointerByte/lock-max/dragon-cmk/proto"
	cmk "github.com/PointerByte/lock-max/dragon-cmk/service/CMK"
	kek "github.com/PointerByte/lock-max/dragon-cmk/service/KEK"
	auth "github.com/PointerByte/lock-max/dragon-cmk/service/auth"
	"google.golang.org/grpc"
)

type fakeGRPCServer struct {
	err                    error
	registerErr            error
	registerCalled         bool
	registeredServiceNames []string
}

func (f *fakeGRPCServer) Serve() error {
	return f.err
}

func (f *fakeGRPCServer) Register(register serverGRPC.RegisterServiceFunc) error {
	f.registerCalled = true
	server := grpc.NewServer()
	register(server)
	for name := range server.GetServiceInfo() {
		f.registeredServiceNames = append(f.registeredServiceNames, name)
	}
	sort.Strings(f.registeredServiceNames)
	return f.registerErr
}

func TestMainStartsServersAndRefreshHooks(t *testing.T) {
	originalNewGRPCServerFn := newGRPCServerFn
	originalNewRepositoriesFn := newRepositoriesFn
	originalNewGRPCServicesFn := newGRPCServicesFn
	originalRegisterRESTRoutesFn := registerRESTRoutesFn
	originalCreateAppFn := createAppFn
	originalStartServerFn := startServerFn
	originalSetFunctionsRefreshFn := setFunctionsRefreshFn
	originalRestartPoolFn := restartPoolFn
	originalStopWorkersFn := stopWorkersFn
	originalRunWorkersFn := runWorkersFn
	originalLoadEnvFilesFn := loadEnvFilesFn
	originalPanicFn := panicFn
	t.Cleanup(func() {
		newGRPCServerFn = originalNewGRPCServerFn
		newRepositoriesFn = originalNewRepositoriesFn
		newGRPCServicesFn = originalNewGRPCServicesFn
		registerRESTRoutesFn = originalRegisterRESTRoutesFn
		createAppFn = originalCreateAppFn
		startServerFn = originalStartServerFn
		setFunctionsRefreshFn = originalSetFunctionsRefreshFn
		restartPoolFn = originalRestartPoolFn
		stopWorkersFn = originalStopWorkersFn
		runWorkersFn = originalRunWorkersFn
		loadEnvFilesFn = originalLoadEnvFilesFn
		panicFn = originalPanicFn
	})

	srv := &http.Server{}
	grpcSrv := &fakeGRPCServer{}
	serverGin.SetModeTest()
	t.Setenv("APP_NAME", "dragon-cmk")
	t.Setenv("APP_VERSION", "test-version")

	started := false
	startedCh := make(chan struct{})
	refreshSet := false
	restRegistered := false
	restartCalled := false
	stopCalled := false
	runCalled := false
	panicFn = func(args ...any) {
		t.Fatalf("unexpected panic call: %v", args)
	}
	loadEnvFilesFn = func() {}
	newGRPCServerFn = func() grpcServer {
		return grpcSrv
	}
	newRepositoriesFn = func() (cmk.IRepository, kek.IRepository, auth.IRepository) {
		return nil, nil, nil
	}
	newGRPCServicesFn = func(cmk.IRepository, auth.IRepository) pb.KeyServiceServer {
		return pb.UnimplementedKeyServiceServer{}
	}
	registerRESTRoutesFn = func(_ cmk.IRepository, _ kek.IRepository, _ ...auth.IRepository) {
		restRegistered = true
	}
	createAppFn = func() (*http.Server, error) {
		return srv, nil
	}
	restartPoolFn = func() error {
		restartCalled = true
		return nil
	}
	stopWorkersFn = func() {
		stopCalled = true
	}
	runWorkersFn = func() {
		runCalled = true
	}
	setFunctionsRefreshFn = func(input ...serverGin.HandlerFunctionsRefresh) {
		refreshSet = true
		if len(input) != 2 {
			t.Fatalf("unexpected refresh handlers length: %d", len(input))
		}
		if err := input[0](); err != nil {
			t.Fatalf("unexpected restartPool error: %v", err)
		}
		if err := input[1](); err != nil {
			t.Fatalf("unexpected restartWorker error: %v", err)
		}
	}
	startServerFn = func(got *http.Server) {
		if got != srv {
			t.Fatalf("unexpected server: %#v", got)
		}
		started = true
		close(startedCh)
	}

	main()
	select {
	case <-startedCh:
	case <-time.After(time.Second):
		t.Fatal("expected HTTP server to start")
	}

	if !started || !refreshSet || !restRegistered || !restartCalled || !stopCalled || !runCalled {
		t.Fatalf("expected all hooks to run, got started=%v refresh=%v rest=%v restart=%v stop=%v run=%v", started, refreshSet, restRegistered, restartCalled, stopCalled, runCalled)
	}
	if !grpcSrv.registerCalled {
		t.Fatal("expected grpc services to be registered")
	}
	expectedServices := []string{"key.v1.KeyService"}
	if len(grpcSrv.registeredServiceNames) != len(expectedServices) {
		t.Fatalf("unexpected registered services: %#v", grpcSrv.registeredServiceNames)
	}
	for i, serviceName := range expectedServices {
		if grpcSrv.registeredServiceNames[i] != serviceName {
			t.Fatalf("unexpected registered services: %#v", grpcSrv.registeredServiceNames)
		}
	}
}

func TestMainPanicsWhenGRPCServeFails(t *testing.T) {
	originalNewGRPCServerFn := newGRPCServerFn
	originalNewRepositoriesFn := newRepositoriesFn
	originalNewGRPCServicesFn := newGRPCServicesFn
	originalRegisterRESTRoutesFn := registerRESTRoutesFn
	originalCreateAppFn := createAppFn
	originalStartServerFn := startServerFn
	originalSetFunctionsRefreshFn := setFunctionsRefreshFn
	originalLoadEnvFilesFn := loadEnvFilesFn
	originalPanicFn := panicFn
	t.Cleanup(func() {
		newGRPCServerFn = originalNewGRPCServerFn
		newRepositoriesFn = originalNewRepositoriesFn
		newGRPCServicesFn = originalNewGRPCServicesFn
		registerRESTRoutesFn = originalRegisterRESTRoutesFn
		createAppFn = originalCreateAppFn
		startServerFn = originalStartServerFn
		setFunctionsRefreshFn = originalSetFunctionsRefreshFn
		loadEnvFilesFn = originalLoadEnvFilesFn
		panicFn = originalPanicFn
	})

	expected := errors.New("grpc error")
	panicCalled := false
	grpcSrv := &fakeGRPCServer{err: expected}
	newGRPCServerFn = func() grpcServer {
		return grpcSrv
	}
	loadEnvFilesFn = func() {}
	newRepositoriesFn = func() (cmk.IRepository, kek.IRepository, auth.IRepository) {
		return nil, nil, nil
	}
	newGRPCServicesFn = func(cmk.IRepository, auth.IRepository) pb.KeyServiceServer {
		return pb.UnimplementedKeyServiceServer{}
	}
	registerRESTRoutesFn = func(cmk.IRepository, kek.IRepository, ...auth.IRepository) {}
	createAppFn = func() (*http.Server, error) {
		return &http.Server{}, nil
	}
	setFunctionsRefreshFn = func(...serverGin.HandlerFunctionsRefresh) {
	}
	startServerFn = func(*http.Server) {
	}
	panicFn = func(args ...any) {
		panicCalled = true
		if len(args) != 1 || args[0] != expected {
			t.Fatalf("unexpected panic args: %v", args)
		}
	}

	main()

	if !panicCalled {
		t.Fatal("expected panicFn to be called")
	}
	if !grpcSrv.registerCalled {
		t.Fatal("expected grpc services to be registered before serve")
	}
}

func TestMainPanicsWhenGRPCRegisterFails(t *testing.T) {
	originalNewGRPCServerFn := newGRPCServerFn
	originalNewRepositoriesFn := newRepositoriesFn
	originalNewGRPCServicesFn := newGRPCServicesFn
	originalRegisterRESTRoutesFn := registerRESTRoutesFn
	originalCreateAppFn := createAppFn
	originalStartServerFn := startServerFn
	originalSetFunctionsRefreshFn := setFunctionsRefreshFn
	originalLoadEnvFilesFn := loadEnvFilesFn
	originalPanicFn := panicFn
	t.Cleanup(func() {
		newGRPCServerFn = originalNewGRPCServerFn
		newRepositoriesFn = originalNewRepositoriesFn
		newGRPCServicesFn = originalNewGRPCServicesFn
		registerRESTRoutesFn = originalRegisterRESTRoutesFn
		createAppFn = originalCreateAppFn
		startServerFn = originalStartServerFn
		setFunctionsRefreshFn = originalSetFunctionsRefreshFn
		loadEnvFilesFn = originalLoadEnvFilesFn
		panicFn = originalPanicFn
	})

	expected := errors.New("register error")
	panicCalled := false
	grpcSrv := &fakeGRPCServer{registerErr: expected}
	newGRPCServerFn = func() grpcServer {
		return grpcSrv
	}
	loadEnvFilesFn = func() {}
	newRepositoriesFn = func() (cmk.IRepository, kek.IRepository, auth.IRepository) {
		return nil, nil, nil
	}
	newGRPCServicesFn = func(cmk.IRepository, auth.IRepository) pb.KeyServiceServer {
		return pb.UnimplementedKeyServiceServer{}
	}
	registerRESTRoutesFn = func(cmk.IRepository, kek.IRepository, ...auth.IRepository) {}
	createAppFn = func() (*http.Server, error) {
		return &http.Server{}, nil
	}
	setFunctionsRefreshFn = func(...serverGin.HandlerFunctionsRefresh) {
	}
	startServerFn = func(*http.Server) {
	}
	panicFn = func(args ...any) {
		panicCalled = true
		if len(args) != 1 || args[0] != expected {
			t.Fatalf("unexpected panic args: %v", args)
		}
	}

	main()

	if !panicCalled {
		t.Fatal("expected panicFn to be called")
	}
	if !grpcSrv.registerCalled {
		t.Fatal("expected grpc register to be called")
	}
}

func TestMainPanicsWhenCreateAppFails(t *testing.T) {
	originalNewGRPCServerFn := newGRPCServerFn
	originalNewRepositoriesFn := newRepositoriesFn
	originalNewGRPCServicesFn := newGRPCServicesFn
	originalRegisterRESTRoutesFn := registerRESTRoutesFn
	originalCreateAppFn := createAppFn
	originalStartServerFn := startServerFn
	originalLoadEnvFilesFn := loadEnvFilesFn
	originalPanicFn := panicFn
	t.Cleanup(func() {
		newGRPCServerFn = originalNewGRPCServerFn
		newRepositoriesFn = originalNewRepositoriesFn
		newGRPCServicesFn = originalNewGRPCServicesFn
		registerRESTRoutesFn = originalRegisterRESTRoutesFn
		createAppFn = originalCreateAppFn
		startServerFn = originalStartServerFn
		loadEnvFilesFn = originalLoadEnvFilesFn
		panicFn = originalPanicFn
	})

	expected := errors.New("app error")
	panicCalled := false
	newGRPCServerFn = func() grpcServer {
		return &fakeGRPCServer{}
	}
	loadEnvFilesFn = func() {}
	newRepositoriesFn = func() (cmk.IRepository, kek.IRepository, auth.IRepository) {
		t.Fatal("newRepositoriesFn should not be called")
		return nil, nil, nil
	}
	newGRPCServicesFn = func(cmk.IRepository, auth.IRepository) pb.KeyServiceServer {
		t.Fatal("newGRPCServicesFn should not be called")
		return nil
	}
	registerRESTRoutesFn = func(cmk.IRepository, kek.IRepository, ...auth.IRepository) {
		t.Fatal("registerRESTRoutesFn should not be called")
	}
	createAppFn = func() (*http.Server, error) {
		return nil, expected
	}
	startServerFn = func(*http.Server) {
		t.Fatal("startServerFn should not be called")
	}
	panicFn = func(args ...any) {
		panicCalled = true
		if len(args) != 1 || args[0] != expected {
			t.Fatalf("unexpected panic args: %v", args)
		}
	}

	main()

	if !panicCalled {
		t.Fatal("expected panicFn to be called")
	}
}

func TestDefaultFunctionVariablesSmoke(t *testing.T) {
	serverGin.SetModeTest()

	if got := newGRPCServerFn(); got == nil {
		t.Fatal("expected default grpc server")
	}

	if got, err := createAppFn(); got == nil && err == nil {
		t.Fatal("expected default app server or error")
	}

	keyRepository, wrappingRepository, authRepository := newRepositoriesFn()
	if keyRepository == nil || wrappingRepository == nil || authRepository == nil {
		t.Fatal("expected default repositories")
	}

	keyService := newGRPCServicesFn(keyRepository, authRepository)
	if keyService == nil {
		t.Fatal("expected default grpc services")
	}

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("expected default panic function to panic")
			}
		}()
		panicFn("boom")
	}()
}
