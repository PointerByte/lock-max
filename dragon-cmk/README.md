# Dragon CMK

Dragon CMK is a Customer Managed Key service with REST and gRPC APIs for key lifecycle, encryption/decryption, signatures, JWT operations, wrapping key configuration, and API token issuance.

## Main APIs

REST listens on `server.gin.port` and gRPC listens on `server.grpc.port`.

Auth endpoint without JWT:

- `POST /api/auth/v1/service-token`: creates a JWT using HTTP Basic auth. Credentials come from `CMK_SERVICE_CLIENT_ID` and `CMK_SERVICE_CLIENT_SECRET`.

Auth endpoints that require `Authorization: Bearer <token>`:

- `POST /api/auth/v1/token`: creates a JWT using a JSON `client_id` and `client_secret` validated against the hashed API client table.
- `POST /api/auth/v1/clients`: creates an API client. `client_id` and `client_secret` are stored as HMAC hashes.
- `GET /api/auth/v1/clients/list?page=1&totalResgisterPage=100`: lists API clients.
- `GET /api/auth/v1/clients/{client_id}`: gets API client metadata.
- `DELETE /api/auth/v1/clients`: deletes an API client using body `{"client_id":"..."}`.

These APIs also require `Authorization: Bearer <token>`:

- `/api/keys/v1/*`: CMK lifecycle, key versions, and creation queues.
- `/api/config/v1/*`: KEK/wrapping key configuration.

The gRPC `key.v1.KeyService` exposes the same auth methods: `CreateServiceToken`, `CreateAPIToken`, `CreateAPIClient`, `ListAPIClients`, `GetAPIClient`, and `DeleteAPIClient`. These methods are excluded from the JWT unary interceptor.

## Configuration

Required PostgreSQL variables:

- `PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`, `PGSSLMODE`. The service user is `dragon_cmk_user`; `PGPASSWORD` must match its password.
- `PG_MAX_CONNS`, `PG_MIN_CONNS`, `PG_MAX_CONN_LIFETIME`, `PG_MAX_CONN_IDLE_TIME`
- `PGADMIN_USER`, `PGADMIN_PASSWORD`: PostgreSQL admin credentials used by `storage/init-db.sh` to create/update the database, schema, and runtime role. `storage/init-db.sh` uses `PGPASSWORD` as the password for `dragon_cmk_user`.

Local environment files:

- `.env`: only `VAULT_MODE` and `WORKERS_LIMIT`.
- `.env.local`: PostgreSQL settings, database passwords, service auth, token TTL, and local KEK secret.

`compose.yaml` loads `.env.local` into the application container and uses `${VAR:-default}` interpolation for declared values. Docker Compose interpolation reads shell variables and `.env`; use `docker compose --env-file .env.local up` or export variables when changing interpolated database defaults.

Auth and JWT variables:

- `CMK_SERVICE_CLIENT_ID`: Basic auth client id for `/service-token`.
- `CMK_SERVICE_CLIENT_SECRET`: Basic auth secret for `/service-token`.
- `CMK_JWT_TTL`: issued token TTL. Default: `1h`.
- `KEK_LOCAL_ENCRYPT_SECRET`: optional secret used when encrypted local KEK private keys are generated/read.
- `WORKERS_LIMIT`: worker pool limit.

JWT signing settings are read from Viper configuration (`application.yaml` under `jwt.*`), not from `CMK_JWT_*` environment variables. For the default EdDSA setup:

```yaml
jwt:
  enable: true
  transport: header
  algorithm: EDDSA
  eddsa:
    private_key: ./certs/jwt/key.pem
    public_key: ./certs/jwt/public.pem
```

## Storage

Database initialization runs:

```bash
storage/init-db.sh
```

New auth objects:

- `dragon_cmk.api_client`
- `dragon_cmk.vw_api_client`
- `dragon_cmk.sp_create_api_client`
- `dragon_cmk.sp_delete_api_client`

API client hashes use `github.com/PointerByte/GoForge/encrypt` HMAC-SHA256. The HMAC secret is the global CMK secret returned by the global KEK.

## OpenTelemetry

GoForge initializes OpenTelemetry when the Gin server is created. HTTP and gRPC middleware are registered by the GoForge server packages, while structured logs are emitted through `github.com/PointerByte/GoForge/logger`.

Enable traces and metrics with standard OTEL variables:

```bash
export OTEL_SDK_DISABLED=false
export OTEL_SERVICE_NAME=dragon-cmk
export OTEL_TRACES_EXPORTER=otlp
export OTEL_METRICS_EXPORTER=otlp
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
export OTEL_PROPAGATORS=tracecontext,baggage
```

For gRPC OTLP:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc
```

For local trace debugging:

```bash
export OTEL_TRACES_EXPORTER=console
export OTEL_METRICS_EXPORTER=none
```

Logs are written through the GoForge logger configuration:

```yaml
logger:
  dir: logs
  level: info
  formatter: json
  rotate:
    enable: true
```

GoForge log export uses the OTLP HTTP log exporter. Configure the same collector endpoint with `OTEL_EXPORTER_OTLP_ENDPOINT`.

## Go Client

The Go client lives in `clients/golang` and uses a Proxy facade over GoForge HTTP and gRPC clients.

```go
package main

import (
	"context"

	dragoncmk "github.com/PointerByte/lock-max/dragon-cmk/clients/golang"
)

func main() {
	ctx := context.Background()

	_ = dragoncmk.Configure(dragoncmk.Config{
		RESTBaseURL: "http://localhost:8080",
		GRPCAddress: "localhost:50051",
	})

	token, err := dragoncmk.RESTCreateServiceToken(ctx, "client-id", "client-secret")
	if err != nil {
		panic(err)
	}

	_ = dragoncmk.Configure(dragoncmk.Config{Token: token.Token})
	_, _ = dragoncmk.GRPCStatus(ctx)
}
```

## Development

Regenerate protobuf:

```bash
buf generate
```

Run tests with writable Go caches when the home cache is read-only:

```bash
GOCACHE=/tmp/go-build GOTMPDIR=/tmp go test ./...
```

Run unit tests and generate an atomic coverage profile:

```bash
go test -cover -covermode=atomic -coverprofile=coverage.out ./...
```
