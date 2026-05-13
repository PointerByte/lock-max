# Dragon CMK

Dragon CMK es un servicio de Customer Managed Keys con APIs REST y gRPC para ciclo de vida de llaves, cifrado/descifrado, firmas, operaciones JWT, configuración de wrapping keys y emisión de tokens para APIs.

## APIs principales

REST escucha en `server.gin.port` y gRPC en `server.grpc.port`.

Endpoint de auth sin JWT:

- `POST /api/auth/v1/service-token`: crea un JWT usando autenticación HTTP Basic. Las credenciales vienen de `CMK_SERVICE_CLIENT_ID` y `CMK_SERVICE_CLIENT_SECRET`.

Endpoints de auth que requieren `Authorization: Bearer <token>`:

- `POST /api/auth/v1/token`: crea un JWT usando `client_id` y `client_secret` en JSON, validados contra la tabla de clientes API hasheados.
- `POST /api/auth/v1/clients`: crea un cliente API. `client_id` y `client_secret` se guardan como hashes HMAC.
- `GET /api/auth/v1/clients/list?page=1&totalResgisterPage=100`: lista clientes API.
- `GET /api/auth/v1/clients/{client_id}`: obtiene metadata de un cliente API.
- `DELETE /api/auth/v1/clients`: elimina un cliente API con body `{"client_id":"..."}`.

Estas APIs también requieren `Authorization: Bearer <token>`:

- `/api/keys/v1/*`: ciclo de vida CMK, versiones de llave y colas de creación.
- `/api/config/v1/*`: configuración KEK/wrapping key.

El servicio gRPC `key.v1.KeyService` expone los métodos de auth equivalentes: `CreateServiceToken`, `CreateAPIToken`, `CreateAPIClient`, `ListAPIClients`, `GetAPIClient` y `DeleteAPIClient`. Estos métodos están excluidos del interceptor JWT.

## Configuración

Variables PostgreSQL:

- `PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`, `PGSSLMODE`. El usuario runtime del servicio es `dragon_cmk_user`; `PGPASSWORD` debe coincidir con su password.
- `PG_MAX_CONNS`, `PG_MIN_CONNS`, `PG_MAX_CONN_LIFETIME`, `PG_MAX_CONN_IDLE_TIME`
- `PGADMIN_USER`, `PGADMIN_PASSWORD`: credenciales admin de PostgreSQL usadas por `storage/init-db.sh` para crear/actualizar la base, schema y rol runtime. `storage/init-db.sh` usa `PGPASSWORD` como password de `dragon_cmk_user`.

Archivos de entorno locales:

- `.env`: solo `VAULT_MODE` y `WORKERS_LIMIT`.
- `.env.local`: configuración PostgreSQL, passwords de base de datos, auth de servicio, TTL de tokens y secreto KEK local.

`compose.yaml` carga `.env.local` en el contenedor de la aplicación y usa interpolación `${VAR:-default}` en los valores declarados. Docker Compose interpola desde variables del shell y `.env`; usa `docker compose --env-file .env.local up` o exporta variables cuando cambies defaults interpolados de base de datos.

Variables de auth y JWT:

- `CMK_SERVICE_CLIENT_ID`: client id de Basic auth para `/service-token`.
- `CMK_SERVICE_CLIENT_SECRET`: secreto de Basic auth para `/service-token`.
- `CMK_JWT_TTL`: TTL de tokens emitidos. Default: `1h`.
- `KEK_LOCAL_ENCRYPT_SECRET`: secreto opcional para generar/leer llaves privadas KEK locales cifradas.
- `WORKERS_LIMIT`: límite del pool de workers.

La configuración de firma JWT se lee desde Viper (`application.yaml` en `jwt.*`), no desde variables de entorno `CMK_JWT_*`. Para EdDSA:

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

Inicializa la base con:

```bash
storage/init-db.sh
```

Nuevos objetos de auth:

- `dragon_cmk.api_client`
- `dragon_cmk.vw_api_client`
- `dragon_cmk.sp_create_api_client`
- `dragon_cmk.sp_delete_api_client`

Los hashes de clientes API usan HMAC-SHA256 de `github.com/PointerByte/GoForge/encrypt`. El secreto HMAC es el secreto CMK global devuelto por la KEK global.

## OpenTelemetry

GoForge inicializa OpenTelemetry al crear el servidor Gin. Los middlewares HTTP y gRPC se registran desde los paquetes de servidor de GoForge, y los logs estructurados se emiten con `github.com/PointerByte/GoForge/logger`.

Activa trazas y métricas con variables estándar OTEL:

```bash
export OTEL_SDK_DISABLED=false
export OTEL_SERVICE_NAME=dragon-cmk
export OTEL_TRACES_EXPORTER=otlp
export OTEL_METRICS_EXPORTER=otlp
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
export OTEL_PROPAGATORS=tracecontext,baggage
```

Para OTLP gRPC:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc
```

Para depurar trazas localmente:

```bash
export OTEL_TRACES_EXPORTER=console
export OTEL_METRICS_EXPORTER=none
```

Los logs se configuran desde GoForge:

```yaml
logger:
  dir: logs
  level: info
  formatter: json
  rotate:
    enable: true
```

La exportación OTLP de logs usa HTTP. Configura el collector con `OTEL_EXPORTER_OTLP_ENDPOINT`.

## Cliente Go

El cliente Go está en `clients/golang` y usa una fachada Proxy sobre clientes HTTP y gRPC de GoForge.

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

## Desarrollo

Regenerar protobuf:

```bash
buf generate
```

Ejecutar pruebas con cachés de Go escribibles cuando el home está en solo lectura:

```bash
GOCACHE=/tmp/go-build GOTMPDIR=/tmp go test ./...
```

Ejecutar pruebas unitarias y generar un perfil de cobertura atómico:

```bash
go test -cover -covermode=atomic -coverprofile=coverage.out ./...
```
