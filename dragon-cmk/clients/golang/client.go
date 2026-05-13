// Copyright 2026 PointerByte Contributors
// SPDX-License-Identifier: Apache-2.0

package dragoncmk

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	clientGRPC "github.com/PointerByte/GoForge/config/client/grpc"
	clientHTTP "github.com/PointerByte/GoForge/config/client/http"
	pb "github.com/PointerByte/lock-max/dragon-cmk/proto"
	"google.golang.org/grpc/metadata"
)

const (
	defaultRESTBaseURL = "http://localhost:8080"
	defaultGRPCAddress = "localhost:50051"
	defaultTimeout     = 30 * time.Second
)

type Config struct {
	RESTBaseURL string
	GRPCAddress string
	Token       string
	Timeout     time.Duration
	HTTPClient  *http.Client
	GRPCClient  clientGRPC.IClient
}

type TokenResponse struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
	ExpiresAt string `json:"expires_at"`
}

type ClientCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type CreateAPIClientRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Description  string `json:"description,omitempty"`
}

type APIClientInfo struct {
	IDAPIClient  string `json:"id_api_client"`
	ClientIDHash string `json:"client_id_hash"`
	Description  string `json:"description,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type Pagination struct {
	TotalRegisters     uint `json:"totalRegisters"`
	TotalPages         uint `json:"totalPages"`
	TotalRegistersPage uint `json:"totalRegistersPage"`
	PageNow            uint `json:"pageNow"`
}

type OperationStatus struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type Proxy interface {
	RESTCreateServiceToken(ctx context.Context, clientID string, clientSecret string) (*TokenResponse, error)
	RESTCreateAPIToken(ctx context.Context, input ClientCredentials) (*TokenResponse, error)
	RESTCreateAPIClient(ctx context.Context, input CreateAPIClientRequest) (*APIClientInfo, error)
	RESTListAPIClients(ctx context.Context, page uint, totalRegisterPage uint) ([]APIClientInfo, *Pagination, error)
	RESTGetAPIClient(ctx context.Context, clientID string) (*APIClientInfo, error)
	RESTDeleteAPIClient(ctx context.Context, clientID string) (*OperationStatus, error)

	GRPCCreateServiceToken(ctx context.Context, clientID string, clientSecret string) (*pb.TokenResponse, error)
	GRPCCreateAPIToken(ctx context.Context, input *pb.CreateAPITokenRequest) (*pb.TokenResponse, error)
	GRPCCreateAPIClient(ctx context.Context, input *pb.CreateAPIClientRequest) (*pb.CreateAPIClientResponse, error)
	GRPCListAPIClients(ctx context.Context, input *pb.ListAPIClientsRequest) (*pb.ListAPIClientsResponse, error)
	GRPCGetAPIClient(ctx context.Context, input *pb.GetAPIClientRequest) (*pb.CreateAPIClientResponse, error)
	GRPCDeleteAPIClient(ctx context.Context, input *pb.DeleteAPIClientRequest) (*pb.OperationStatus, error)
	GRPCStatus(ctx context.Context) (*pb.StatusResponse, error)
	GRPCKeyServiceClient(ctx context.Context) (pb.KeyServiceClient, error)
	Close() error
}

type proxy struct {
	rest *restProxy
	grpc *grpcProxy
}

type restProxy struct {
	baseURL string
	token   string
	client  *http.Client
}

type grpcProxy struct {
	token  string
	client clientGRPC.IClient
}

type restEnvelope struct {
	ServiceName    string          `json:"serviceName"`
	ServiceVersion string          `json:"serviceVersion"`
	Results        json.RawMessage `json:"results,omitempty"`
	Pagination     *Pagination     `json:"pagination,omitempty"`
}

type restError struct {
	Error string `json:"error"`
}

type apiClientResult struct {
	APIClient APIClientInfo `json:"api_client"`
}

type operationResult struct {
	Status OperationStatus `json:"status"`
}

var (
	defaultProxyMu sync.RWMutex
	defaultProxy   Proxy = MustNewProxy(Config{})
)

func MustNewProxy(config Config) Proxy {
	client, err := NewProxy(config)
	if err != nil {
		panic(err)
	}
	return client
}

func NewProxy(config Config) (Proxy, error) {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = clientHTTP.NewRestClient(timeout, nil)
	}

	grpcClient := config.GRPCClient
	if grpcClient == nil {
		grpcClient = clientGRPC.NewIClient(nil)
		grpcClient.SetAddress(valueOrDefault(config.GRPCAddress, defaultGRPCAddress))
	}

	return &proxy{
		rest: &restProxy{
			baseURL: strings.TrimRight(valueOrDefault(config.RESTBaseURL, defaultRESTBaseURL), "/"),
			token:   strings.TrimSpace(config.Token),
			client:  httpClient,
		},
		grpc: &grpcProxy{
			token:  strings.TrimSpace(config.Token),
			client: grpcClient,
		},
	}, nil
}

func Configure(config Config) error {
	client, err := NewProxy(config)
	if err != nil {
		return err
	}
	SetProxy(client)
	return nil
}

func SetProxy(client Proxy) {
	if client == nil {
		return
	}
	defaultProxyMu.Lock()
	defer defaultProxyMu.Unlock()
	defaultProxy = client
}

func DefaultProxy() Proxy {
	defaultProxyMu.RLock()
	defer defaultProxyMu.RUnlock()
	return defaultProxy
}

func Close() error {
	return DefaultProxy().Close()
}

func RESTCreateServiceToken(ctx context.Context, clientID string, clientSecret string) (*TokenResponse, error) {
	return DefaultProxy().RESTCreateServiceToken(ctx, clientID, clientSecret)
}

func RESTCreateAPIToken(ctx context.Context, input ClientCredentials) (*TokenResponse, error) {
	return DefaultProxy().RESTCreateAPIToken(ctx, input)
}

func RESTCreateAPIClient(ctx context.Context, input CreateAPIClientRequest) (*APIClientInfo, error) {
	return DefaultProxy().RESTCreateAPIClient(ctx, input)
}

func RESTListAPIClients(ctx context.Context, page uint, totalRegisterPage uint) ([]APIClientInfo, *Pagination, error) {
	return DefaultProxy().RESTListAPIClients(ctx, page, totalRegisterPage)
}

func RESTGetAPIClient(ctx context.Context, clientID string) (*APIClientInfo, error) {
	return DefaultProxy().RESTGetAPIClient(ctx, clientID)
}

func RESTDeleteAPIClient(ctx context.Context, clientID string) (*OperationStatus, error) {
	return DefaultProxy().RESTDeleteAPIClient(ctx, clientID)
}

func GRPCCreateServiceToken(ctx context.Context, clientID string, clientSecret string) (*pb.TokenResponse, error) {
	return DefaultProxy().GRPCCreateServiceToken(ctx, clientID, clientSecret)
}

func GRPCCreateAPIToken(ctx context.Context, input *pb.CreateAPITokenRequest) (*pb.TokenResponse, error) {
	return DefaultProxy().GRPCCreateAPIToken(ctx, input)
}

func GRPCCreateAPIClient(ctx context.Context, input *pb.CreateAPIClientRequest) (*pb.CreateAPIClientResponse, error) {
	return DefaultProxy().GRPCCreateAPIClient(ctx, input)
}

func GRPCListAPIClients(ctx context.Context, input *pb.ListAPIClientsRequest) (*pb.ListAPIClientsResponse, error) {
	return DefaultProxy().GRPCListAPIClients(ctx, input)
}

func GRPCGetAPIClient(ctx context.Context, input *pb.GetAPIClientRequest) (*pb.CreateAPIClientResponse, error) {
	return DefaultProxy().GRPCGetAPIClient(ctx, input)
}

func GRPCDeleteAPIClient(ctx context.Context, input *pb.DeleteAPIClientRequest) (*pb.OperationStatus, error) {
	return DefaultProxy().GRPCDeleteAPIClient(ctx, input)
}

func GRPCStatus(ctx context.Context) (*pb.StatusResponse, error) {
	return DefaultProxy().GRPCStatus(ctx)
}

func GRPCKeyServiceClient(ctx context.Context) (pb.KeyServiceClient, error) {
	return DefaultProxy().GRPCKeyServiceClient(ctx)
}

func (p *proxy) RESTCreateServiceToken(ctx context.Context, clientID string, clientSecret string) (*TokenResponse, error) {
	return p.rest.createServiceToken(ctx, clientID, clientSecret)
}

func (p *proxy) RESTCreateAPIToken(ctx context.Context, input ClientCredentials) (*TokenResponse, error) {
	return p.rest.createAPIToken(ctx, input)
}

func (p *proxy) RESTCreateAPIClient(ctx context.Context, input CreateAPIClientRequest) (*APIClientInfo, error) {
	return p.rest.createAPIClient(ctx, input)
}

func (p *proxy) RESTListAPIClients(ctx context.Context, page uint, totalRegisterPage uint) ([]APIClientInfo, *Pagination, error) {
	return p.rest.listAPIClients(ctx, page, totalRegisterPage)
}

func (p *proxy) RESTGetAPIClient(ctx context.Context, clientID string) (*APIClientInfo, error) {
	return p.rest.getAPIClient(ctx, clientID)
}

func (p *proxy) RESTDeleteAPIClient(ctx context.Context, clientID string) (*OperationStatus, error) {
	return p.rest.deleteAPIClient(ctx, clientID)
}

func (p *proxy) GRPCCreateServiceToken(ctx context.Context, clientID string, clientSecret string) (*pb.TokenResponse, error) {
	return p.grpc.createServiceToken(ctx, clientID, clientSecret)
}

func (p *proxy) GRPCCreateAPIToken(ctx context.Context, input *pb.CreateAPITokenRequest) (*pb.TokenResponse, error) {
	client, err := p.grpc.keyService(ctx)
	if err != nil {
		return nil, err
	}
	return client.CreateAPIToken(p.grpc.bearerContext(ctx), input)
}

func (p *proxy) GRPCCreateAPIClient(ctx context.Context, input *pb.CreateAPIClientRequest) (*pb.CreateAPIClientResponse, error) {
	client, err := p.grpc.keyService(ctx)
	if err != nil {
		return nil, err
	}
	return client.CreateAPIClient(p.grpc.bearerContext(ctx), input)
}

func (p *proxy) GRPCListAPIClients(ctx context.Context, input *pb.ListAPIClientsRequest) (*pb.ListAPIClientsResponse, error) {
	client, err := p.grpc.keyService(ctx)
	if err != nil {
		return nil, err
	}
	return client.ListAPIClients(p.grpc.bearerContext(ctx), input)
}

func (p *proxy) GRPCGetAPIClient(ctx context.Context, input *pb.GetAPIClientRequest) (*pb.CreateAPIClientResponse, error) {
	client, err := p.grpc.keyService(ctx)
	if err != nil {
		return nil, err
	}
	return client.GetAPIClient(p.grpc.bearerContext(ctx), input)
}

func (p *proxy) GRPCDeleteAPIClient(ctx context.Context, input *pb.DeleteAPIClientRequest) (*pb.OperationStatus, error) {
	client, err := p.grpc.keyService(ctx)
	if err != nil {
		return nil, err
	}
	return client.DeleteAPIClient(p.grpc.bearerContext(ctx), input)
}

func (p *proxy) GRPCStatus(ctx context.Context) (*pb.StatusResponse, error) {
	client, err := p.grpc.keyService(ctx)
	if err != nil {
		return nil, err
	}
	return client.Status(p.grpc.bearerContext(ctx), &pb.StatusRequest{})
}

func (p *proxy) GRPCKeyServiceClient(ctx context.Context) (pb.KeyServiceClient, error) {
	return p.grpc.keyService(ctx)
}

func (p *proxy) Close() error {
	return p.grpc.client.Close()
}

func (r *restProxy) createServiceToken(ctx context.Context, clientID string, clientSecret string) (*TokenResponse, error) {
	var output TokenResponse
	headers := http.Header{}
	headers.Set("Authorization", basicAuthorization(clientID, clientSecret))
	return &output, r.doJSON(ctx, http.MethodPost, "/api/auth/v1/service-token", headers, nil, &output)
}

func (r *restProxy) createAPIToken(ctx context.Context, input ClientCredentials) (*TokenResponse, error) {
	var output TokenResponse
	return &output, r.doJSON(ctx, http.MethodPost, "/api/auth/v1/token", nil, input, &output)
}

func (r *restProxy) createAPIClient(ctx context.Context, input CreateAPIClientRequest) (*APIClientInfo, error) {
	var output apiClientResult
	if err := r.doJSON(ctx, http.MethodPost, "/api/auth/v1/clients", nil, input, &output); err != nil {
		return nil, err
	}
	return &output.APIClient, nil
}

func (r *restProxy) listAPIClients(ctx context.Context, page uint, totalRegisterPage uint) ([]APIClientInfo, *Pagination, error) {
	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("totalResgisterPage", fmt.Sprintf("%d", totalRegisterPage))

	var output []APIClientInfo
	pagination, err := r.doJSONWithPagination(ctx, http.MethodGet, "/api/auth/v1/clients/list?"+params.Encode(), nil, nil, &output)
	return output, pagination, err
}

func (r *restProxy) getAPIClient(ctx context.Context, clientID string) (*APIClientInfo, error) {
	var output apiClientResult
	path := "/api/auth/v1/clients/" + url.PathEscape(clientID)
	if err := r.doJSON(ctx, http.MethodGet, path, nil, nil, &output); err != nil {
		return nil, err
	}
	return &output.APIClient, nil
}

func (r *restProxy) deleteAPIClient(ctx context.Context, clientID string) (*OperationStatus, error) {
	var output operationResult
	if err := r.doJSON(ctx, http.MethodDelete, "/api/auth/v1/clients", nil, map[string]string{"client_id": clientID}, &output); err != nil {
		return nil, err
	}
	return &output.Status, nil
}

func (r *restProxy) doJSON(ctx context.Context, method string, path string, headers http.Header, body any, output any) error {
	_, err := r.doJSONWithPagination(ctx, method, path, headers, body, output)
	return err
}

func (r *restProxy) doJSONWithPagination(ctx context.Context, method string, path string, headers http.Header, body any, output any) (*Pagination, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, r.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if r.token != "" {
		request.Header.Set("Authorization", "Bearer "+r.token)
	}
	for key, values := range headers {
		for _, value := range values {
			request.Header.Set(key, value)
		}
	}

	response, err := r.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var envelope restEnvelope
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, &envelope); err != nil {
			return nil, err
		}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, responseError(response.StatusCode, envelope.Results)
	}
	if output != nil && len(envelope.Results) > 0 {
		if err := json.Unmarshal(envelope.Results, output); err != nil {
			return nil, err
		}
	}
	return envelope.Pagination, nil
}

func (g *grpcProxy) createServiceToken(ctx context.Context, clientID string, clientSecret string) (*pb.TokenResponse, error) {
	client, err := g.keyService(ctx)
	if err != nil {
		return nil, err
	}
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", basicAuthorization(clientID, clientSecret))
	return client.CreateServiceToken(ctx, &pb.CreateServiceTokenRequest{})
}

func (g *grpcProxy) keyService(ctx context.Context) (pb.KeyServiceClient, error) {
	if ctx != nil {
		g.client.SetContext(ctx)
	}
	return clientGRPC.BuildClient(g.client, pb.NewKeyServiceClient)
}

func (g *grpcProxy) bearerContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if g.token == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+g.token)
}

func responseError(statusCode int, raw json.RawMessage) error {
	var apiError restError
	if len(raw) > 0 && json.Unmarshal(raw, &apiError) == nil && apiError.Error != "" {
		return fmt.Errorf("dragon-cmk rest status %d: %s", statusCode, apiError.Error)
	}
	return fmt.Errorf("dragon-cmk rest status %d", statusCode)
}

func basicAuthorization(clientID string, clientSecret string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret))
}

func valueOrDefault(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
