package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	serverGin "github.com/PointerByte/GoForge/config/server/gin"
	jwtservice "github.com/PointerByte/GoForge/security/auth/jwt"
	commonEntity "github.com/PointerByte/lock-max/dragon-cmk/entity/common"
	"github.com/PointerByte/lock-max/dragon-cmk/service/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

const restTestJWTSecret = "rest-test-secret"

var restTestBearerToken string

func newRESTTestRouter(t *testing.T, keyRepository *fakeCMKRepository, wrappingRepository *fakeWrappingRepository) *gin.Engine {
	t.Helper()

	originalGetRouteFn := getRouteFn
	originalJWTEnable := viper.Get("jwt.enable")
	originalJWTAlgorithm := viper.Get("jwt.algorithm")
	originalJWTSecret := viper.Get("jwt.hmac.secret")
	t.Cleanup(func() {
		getRouteFn = originalGetRouteFn
		restoreViperValue("jwt.enable", originalJWTEnable)
		restoreViperValue("jwt.algorithm", originalJWTAlgorithm)
		restoreViperValue("jwt.hmac.secret", originalJWTSecret)
		restTestBearerToken = ""
	})

	serverGin.SetModeTest()
	viper.Set("jwt.enable", true)
	viper.Set("jwt.algorithm", "HS256")
	viper.Set("jwt.hmac.secret", restTestJWTSecret)
	restTestBearerToken = newRESTTestBearerToken(t)

	router := gin.New()
	getRouteFn = func(relativePath string) *gin.RouterGroup {
		return router.Group(relativePath)
	}
	RegisterRESTRoutes(keyRepository, wrappingRepository)
	return router
}

func restoreViperValue(key string, value any) {
	if value == nil {
		viper.Set(key, nil)
		return
	}
	viper.Set(key, value)
}

func newRESTTestBearerToken(t *testing.T) string {
	t.Helper()

	service, err := jwtservice.NewHMACService(jwtservice.HMACServiceInput{Secret: restTestJWTSecret})
	if err != nil {
		t.Fatalf("create jwt service: %v", err)
	}
	token, err := service.Create(map[string]any{"sub": "rest-test"})
	if err != nil {
		t.Fatalf("create jwt token: %v", err)
	}
	return "Bearer " + token
}

func performJSON(router *gin.Engine, method string, path string, body any) *httptest.ResponseRecorder {
	payload := bytes.NewBuffer(nil)
	if body != nil {
		_ = json.NewEncoder(payload).Encode(body)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, payload)
	request.Header.Set("Content-Type", "application/json")
	if restTestBearerToken != "" {
		request.Header.Set("Authorization", restTestBearerToken)
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestRESTKeyLifecycleDelegatesToCMK(t *testing.T) {
	idKey := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	idVersion := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	secret := idKey.String() + "." + idVersion.String()
	repository := &fakeCMKRepository{}
	router := newRESTTestRouter(t, repository, &fakeWrappingRepository{})

	tests := []struct {
		name       string
		method     string
		path       string
		body       map[string]any
		wantStatus int
		assert     func(t *testing.T, body map[string]any)
	}{
		{
			name:       "enable",
			method:     http.MethodPost,
			path:       "/api/keys/v1/enable",
			body:       map[string]any{"key_id": idKey.String(), "version_id": idVersion.String()},
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, body map[string]any) {
				if repository.enableSecret != secret || body["key"].(map[string]any)["state"] != "enabled" {
					t.Fatalf("unexpected enable result: body=%#v repo=%#v", body, repository)
				}
			},
		},
		{
			name:       "disable",
			method:     http.MethodPost,
			path:       "/api/keys/v1/disable",
			body:       map[string]any{"secret_cmk_key": secret},
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, body map[string]any) {
				if repository.disableSecret != secret || body["key"].(map[string]any)["state"] != "disabled" {
					t.Fatalf("unexpected disable result: body=%#v repo=%#v", body, repository)
				}
			},
		},
		{
			name:       "schedule deletion",
			method:     http.MethodPost,
			path:       "/api/keys/v1/schedule-deletion",
			body:       map[string]any{"key_id": idKey.String(), "version_id": idVersion.String(), "pending_window_days": 7},
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, body map[string]any) {
				if repository.scheduleSecret != secret || repository.scheduleInterval != 7 || body["key"].(map[string]any)["state"] != "pendingDeletion" {
					t.Fatalf("unexpected schedule result: body=%#v repo=%#v", body, repository)
				}
			},
		},
		{
			name:       "pending deletion",
			method:     http.MethodPost,
			path:       "/api/keys/v1/pending-deletion",
			body:       map[string]any{"secret_cmk_key": secret},
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, body map[string]any) {
				if len(repository.pendingSecrets) != 1 || repository.pendingSecrets[0] != secret || body["key"].(map[string]any)["state"] != "pendingDeletion" {
					t.Fatalf("unexpected pending result: body=%#v repo=%#v", body, repository)
				}
			},
		},
		{
			name:       "cancel deletion",
			method:     http.MethodPost,
			path:       "/api/keys/v1/cancel-deletion",
			body:       map[string]any{"secret_cmk_key": secret},
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, body map[string]any) {
				if repository.cancelSecret != secret || body["key"].(map[string]any)["state"] != "disabled" {
					t.Fatalf("unexpected cancel result: body=%#v repo=%#v", body, repository)
				}
			},
		},
		{
			name:       "unavailable",
			method:     http.MethodPost,
			path:       "/api/keys/v1/unavailable",
			body:       map[string]any{"secret_cmk_key": secret},
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, body map[string]any) {
				if repository.unavailableSecret != secret || body["key"].(map[string]any)["state"] != "Unavailable" {
					t.Fatalf("unexpected unavailable result: body=%#v repo=%#v", body, repository)
				}
			},
		},
		{
			name:       "delete",
			method:     http.MethodDelete,
			path:       "/api/keys/v1/",
			body:       map[string]any{"request": map[string]any{"secret_cmk_key": secret}},
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, body map[string]any) {
				if repository.deleteSecret != secret || !body["status"].(map[string]any)["success"].(bool) {
					t.Fatalf("unexpected delete result: body=%#v repo=%#v", body, repository)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := performJSON(router, tt.method, tt.path, tt.body)
			if recorder.Code != tt.wantStatus {
				t.Fatalf("unexpected status code: %d body=%s", recorder.Code, recorder.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("unexpected response body: %v", err)
			}
			if results, ok := body["results"].(map[string]any); ok {
				body = results
			}
			tt.assert(t, body)
		})
	}
}

func TestRESTKeyVersionEndpoints(t *testing.T) {
	idKey := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	idVersion := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	versionInfo := &models.KeyVersionInfo{
		IDCmkKey:        idKey,
		IDCmkKeyVersion: idVersion,
		SecretCmkKey:    "secret",
		VersionNumber:   2,
		Size:            256,
		PublicKey:       "public-key",
		Status:          commonEntity.KeyVersionStatusDisabled,
		Algorithm:       commonEntity.KeySymmetricDefault,
		Purpose:         commonEntity.KeyPurposeEncrypt,
	}
	repository := &fakeCMKRepository{getVersionResult: versionInfo, updateVersionResult: versionInfo}
	router := newRESTTestRouter(t, repository, &fakeWrappingRepository{})

	recorder := performJSON(router, http.MethodGet, "/api/keys/v1/versions/"+idVersion.String(), nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected get version status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if repository.getVersionID != idVersion {
		t.Fatalf("unexpected get version id: %s", repository.getVersionID)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unexpected get response body: %v", err)
	}
	keyVersion := body["results"].(map[string]any)["key_version"].(map[string]any)
	if keyVersion["secret_cmk_key"] != "secret" || keyVersion["public_key"] != "public-key" {
		t.Fatalf("unexpected key version response: %#v", keyVersion)
	}
	if _, exists := keyVersion["secret_wrapped"]; exists {
		t.Fatalf("secret_wrapped must not be exposed: %#v", keyVersion)
	}

	versionInfo.Status = commonEntity.KeyVersionStatusUnavailable
	recorder = performJSON(router, http.MethodPatch, "/api/keys/v1/versions/"+idVersion.String()+"/status", map[string]any{"status": "Unavailable"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected update version status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if repository.updateVersionID != idVersion || repository.updateVersionStatus != "Unavailable" {
		t.Fatalf("unexpected update version input: %#v", repository)
	}

	recorder = performJSON(router, http.MethodPatch, "/api/keys/v1/versions/"+idVersion.String()+"/status", map[string]any{"status": "pendingDeletion"})
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected pendingDeletion validation error, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRESTKEKDelegatesToRepository(t *testing.T) {
	idGenerate := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	idWrap := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	wrappingKey := &models.KEK{
		IdCmkWrappingKeyRef: idWrap,
		Provider:            "local",
		KeyRef:              "local",
		Version:             "v1",
		PublicKey:           "public",
	}
	repository := &fakeWrappingRepository{getData: wrappingKey, createResult: idWrap, rotateResult: idWrap}
	router := newRESTTestRouter(t, &fakeCMKRepository{}, repository)

	recorder := performJSON(router, http.MethodPost, "/api/config/v1/create", map[string]any{
		"id_generate": idGenerate.String(),
		"salt":        "salt",
	})
	if recorder.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if repository.createID != idGenerate || repository.createSecret != "" || repository.createSalt != "salt" {
		t.Fatalf("unexpected create input: %#v", repository)
	}

	recorder = performJSON(router, http.MethodGet, "/api/config/v1/"+idGenerate.String()+"?version=v1", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected get status: %d body=%s", recorder.Code, recorder.Body.String())
	}

	repository.getData = nil
	recorder = performJSON(router, http.MethodGet, "/api/config/v1/"+idGenerate.String()+"?version=v2", nil)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected get missing status 204, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	repository.getData = wrappingKey

	recorder = performJSON(router, http.MethodPost, "/api/config/v1/rotate", map[string]any{"id_generate": idGenerate.String(), "salt": "salt-2"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected rotate status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if repository.rotateID != idGenerate || repository.rotateSalt != "salt-2" {
		t.Fatalf("unexpected rotate input: %#v", repository)
	}

	recorder = performJSON(router, http.MethodDelete, "/api/config/v1/", map[string]any{"id_kek": idWrap.String(), "version": "v1"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected delete status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if repository.deleteID != idWrap || repository.deleteVersion != "v1" {
		t.Fatalf("unexpected delete input: %#v", repository)
	}
}

func TestRESTListEndpointsUseGetQueryParams(t *testing.T) {
	idGenerate := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	idWrap := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	idKey := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	wrappingRepository := &fakeWrappingRepository{
		getData: &models.KEK{
			IdCmkWrappingKeyRef: idWrap,
			Provider:            "local",
			KeyRef:              "local",
			Version:             "v1",
			PublicKey:           "public",
		},
	}
	keyRepository := &fakeCMKRepository{}
	router := newRESTTestRouter(t, keyRepository, wrappingRepository)

	recorder := performJSON(router, http.MethodGet, "/api/config/v1/list?id_generate="+idGenerate.String()+"&page=2&totalResgisterPage=25", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected config list status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if wrappingRepository.listID != idGenerate || wrappingRepository.listPage != 2 || wrappingRepository.listTotalPage != 25 {
		t.Fatalf("unexpected config list input: %#v", wrappingRepository)
	}
	if recorder := performJSON(router, http.MethodPost, "/api/config/v1/list", map[string]any{"id_generate": idGenerate.String(), "page": 2, "totalResgisterPage": 25}); recorder.Code != http.StatusNotFound {
		t.Fatalf("expected config list POST to be unavailable, got %d", recorder.Code)
	}

	recorder = performJSON(router, http.MethodGet, "/api/keys/v1/list?id_kek="+idWrap.String()+"&page=3&totalResgisterPage=50", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected keys list status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if keyRepository.listIDKek != idWrap || keyRepository.listPage != 3 || keyRepository.listTotalPage != 50 {
		t.Fatalf("unexpected keys list input: %#v", keyRepository)
	}
	if recorder := performJSON(router, http.MethodPost, "/api/keys/v1/list", map[string]any{"id_kek": idWrap.String(), "page": 3, "totalResgisterPage": 50}); recorder.Code != http.StatusNotFound {
		t.Fatalf("expected keys list POST to be unavailable, got %d", recorder.Code)
	}

	recorder = performJSON(router, http.MethodGet, "/api/keys/v1/creation-queues/list?id_cmk_key="+idKey.String()+"&status=pending&page=4&totalResgisterPage=15", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected creation queues list status: %d body=%s", recorder.Code, recorder.Body.String())
	}
	if keyRepository.listQueueIDKey != idKey || string(keyRepository.listQueueStatus) != "pending" || keyRepository.listQueuePage != 4 || keyRepository.listQueueTotal != 15 {
		t.Fatalf("unexpected creation queues list input: %#v", keyRepository)
	}
	if recorder := performJSON(router, http.MethodPost, "/api/keys/v1/creation-queues/list", map[string]any{"page": 4, "totalResgisterPage": 15}); recorder.Code != http.StatusNotFound {
		t.Fatalf("expected creation queues list POST to be unavailable, got %d", recorder.Code)
	}
}

func TestRESTValidationAndErrors(t *testing.T) {
	router := newRESTTestRouter(
		t,
		&fakeCMKRepository{enableErr: errors.New("enable failed")},
		&fakeWrappingRepository{getErr: errors.New("get failed")},
	)

	if recorder := performJSON(router, http.MethodPost, "/api/keys/v1/enable", map[string]any{"secret_cmk_key": "secret"}); recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected key service error, got %d", recorder.Code)
	}
	if recorder := performJSON(router, http.MethodPost, "/api/keys/v1/schedule-deletion", map[string]any{"secret_cmk_key": "secret", "pending_window_days": 0}); recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected schedule validation error, got %d", recorder.Code)
	}
	if recorder := performJSON(router, http.MethodPost, "/api/keys/v1/disable", map[string]any{}); recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected missing secret error, got %d", recorder.Code)
	}
	if recorder := performJSON(router, http.MethodPost, "/api/keys/v1/enable?secret_cmk_key=secret", nil); recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected key lifecycle query input to be rejected, got %d", recorder.Code)
	}
	if recorder := performJSON(router, http.MethodDelete, "/api/keys/v1/?secret_cmk_key=secret", nil); recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected delete key query input to be rejected, got %d", recorder.Code)
	}
	if recorder := performJSON(router, http.MethodGet, "/api/config/v1/bad-id", nil); recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid kek id error, got %d", recorder.Code)
	}
	if recorder := performJSON(router, http.MethodGet, "/api/keys/v1/creation-queues/list?status=bad&page=1&totalResgisterPage=10", nil); recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid queue status error, got %d", recorder.Code)
	}
	if recorder := performJSON(router, http.MethodGet, "/api/config/v1/33333333-3333-3333-3333-333333333333", nil); recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected kek service error, got %d", recorder.Code)
	}
}

func TestRESTSwaggerDocsAreInteractive(t *testing.T) {
	router := newRESTTestRouter(t, &fakeCMKRepository{}, &fakeWrappingRepository{})

	recorder := performJSON(router, http.MethodGet, "/api/docs/index.html", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected swagger UI, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("unexpected swagger UI content type: %q", contentType)
	}

	recorder = performJSON(router, http.MethodGet, "/api/docs/doc.json", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected swagger spec, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected swagger spec json: %v", err)
	}
	if body["swagger"] != "2.0" || body["basePath"] != "/" {
		t.Fatalf("unexpected swagger spec body: %#v", body)
	}
}
