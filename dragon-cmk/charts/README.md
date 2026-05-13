# dragon-cmk Helm chart

This chart deploys the `dragon-cmk` service. It can also deploy a PostgreSQL
database in the same release when explicitly enabled.

By default, the chart creates the `lock-max` namespace and deploys namespaced
resources there:

```yaml
namespace:
  create: true
  name: lock-max
```

With the default values, the chart renders a `Namespace` object. Install with:

```bash
helm install dragon-cmk ./charts
```

If you prefer Helm release metadata to live in `lock-max`, let Helm create the
namespace and disable the chart namespace object:

```bash
helm install dragon-cmk ./charts \
  --namespace lock-max \
  --create-namespace \
  --set namespace.create=false
```

## cert-manager

The chart can create cert-manager resources when cert-manager and its CRDs are
already installed in the cluster. It does not install cert-manager.

Enable the integration with:

```bash
helm install dragon-cmk ./charts \
  --set certManager.enabled=true
```

With the default cert-manager values, the chart creates:

1. a self-signed bootstrap `Issuer`
2. a CA `Certificate`
3. a CA-backed `Issuer`
4. a server `Certificate` for the `dragon-cmk` Kubernetes service DNS names
5. a public/client `Certificate` mounted under `/app/certs/public`
6. an Ed25519 `Certificate` used to prepare JWT signing keys under `/app/certs/jwt`

The server certificate is mounted at `/app/certs/server` as:

```text
cert.pem -> tls.crt
key.pem  -> tls.key
ca.pem   -> ca.crt
```

That matches the paths in `application.yaml`.

The public certificate is mounted at `/app/certs/public` with the same file
names:

```text
cert.pem -> tls.crt
key.pem  -> tls.key
ca.pem   -> ca.crt
```

The JWT certificate uses cert-manager `privateKey.algorithm: Ed25519`. The
deployment prepares the files expected by `application.yaml` before the app
starts. The init container uses the same application image and requires
`openssl` to be present:

```text
key.pem    -> Ed25519 private key from tls.key
public.pem -> Ed25519 public key extracted from tls.crt
```

To create a client certificate for another service or an ingress that connects
to `dragon-cmk` with mTLS:

```bash
helm upgrade --install dragon-cmk ./charts \
  --set certManager.enabled=true \
  --set certManager.client.create=true \
  --set certManager.client.commonName=dragon-cmk-client
```

The client certificate is stored in the generated client Secret:

```text
dragon-cmk-client-tls
```

### Service-to-service mTLS

For another pod to call `dragon-cmk` directly, enable the server certificate and
create a client certificate:

```bash
helm upgrade --install dragon-cmk ./charts \
  --set certManager.enabled=true \
  --set certManager.client.create=true \
  --set certManager.client.commonName=orders-api.lock-max
```

Mount the generated client Secret in the caller pod:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: orders-api
  namespace: lock-max
spec:
  template:
    spec:
      containers:
        - name: orders-api
          image: orders-api:latest
          volumeMounts:
            - name: dragon-cmk-client-tls
              mountPath: /etc/dragon-cmk/client
              readOnly: true
      volumes:
        - name: dragon-cmk-client-tls
          secret:
            secretName: dragon-cmk-client-tls
```

Use these files from the caller:

```text
/etc/dragon-cmk/client/tls.crt  client certificate
/etc/dragon-cmk/client/tls.key  client private key
/etc/dragon-cmk/client/ca.crt   CA to verify dragon-cmk
```

The caller should connect to the service DNS name present in the server
certificate SAN:

```text
https://dragon-cmk.lock-max.svc.cluster.local:8080
```

For gRPC, use the same CA and client certificate pair, but connect to port
`50051`.

### ingress-nginx

There are two common patterns with ingress-nginx.

If the browser or external client does mTLS with the ingress, and the ingress
then calls `dragon-cmk`, configure client certificate authentication on the
Ingress:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: dragon-cmk
  namespace: lock-max
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/auth-tls-secret: lock-max/dragon-cmk-ca
    nginx.ingress.kubernetes.io/auth-tls-verify-client: "on"
    nginx.ingress.kubernetes.io/auth-tls-verify-depth: "1"
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - dragon-cmk.example.com
      secretName: dragon-cmk-ingress-tls
  rules:
    - host: dragon-cmk.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: dragon-cmk
                port:
                  name: rest
```

If ingress-nginx must connect to `dragon-cmk` over HTTPS and present a client
certificate to the backend, use backend TLS annotations:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: dragon-cmk
  namespace: lock-max
  annotations:
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
    nginx.ingress.kubernetes.io/proxy-ssl-secret: lock-max/dragon-cmk-client-tls
    nginx.ingress.kubernetes.io/proxy-ssl-verify: "on"
    nginx.ingress.kubernetes.io/proxy-ssl-verify-depth: "1"
    nginx.ingress.kubernetes.io/proxy-ssl-server-name: "on"
    nginx.ingress.kubernetes.io/proxy-ssl-name: dragon-cmk.lock-max.svc.cluster.local
spec:
  ingressClassName: nginx
  rules:
    - host: dragon-cmk.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: dragon-cmk
                port:
                  name: rest
```

`proxy-ssl-secret` must reference a Secret containing `tls.crt`, `tls.key`, and
`ca.crt`. The `dragon-cmk-client-tls` Secret generated by this chart matches
that shape.

### Traefik

For Traefik CRDs, use a `ServersTransport` when Traefik connects to the
`dragon-cmk` backend with HTTPS or mTLS:

```yaml
apiVersion: traefik.io/v1alpha1
kind: ServersTransport
metadata:
  name: dragon-cmk-mtls
  namespace: lock-max
spec:
  serverName: dragon-cmk.lock-max.svc.cluster.local
  rootCAsSecrets:
    - dragon-cmk-ca
  certificatesSecrets:
    - dragon-cmk-client-tls
---
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: dragon-cmk
  namespace: lock-max
spec:
  entryPoints:
    - websecure
  routes:
    - match: Host(`dragon-cmk.example.com`)
      kind: Rule
      services:
        - name: dragon-cmk
          port: 8080
          scheme: https
          serversTransport: dragon-cmk-mtls
  tls:
    secretName: dragon-cmk-ingress-tls
```

To require client certificates at the Traefik edge, add a `TLSOption` and
reference it from the `IngressRoute`:

```yaml
apiVersion: traefik.io/v1alpha1
kind: TLSOption
metadata:
  name: dragon-cmk-client-auth
  namespace: lock-max
spec:
  clientAuth:
    secretNames:
      - dragon-cmk-ca
    clientAuthType: RequireAndVerifyClientCert
---
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: dragon-cmk
  namespace: lock-max
spec:
  entryPoints:
    - websecure
  routes:
    - match: Host(`dragon-cmk.example.com`)
      kind: Rule
      services:
        - name: dragon-cmk
          port: 8080
          scheme: https
          serversTransport: dragon-cmk-mtls
  tls:
    secretName: dragon-cmk-ingress-tls
    options:
      name: dragon-cmk-client-auth
      namespace: lock-max
```

`rootCAsSecrets` verifies the `dragon-cmk` server certificate, and
`certificatesSecrets` is the client certificate Traefik presents to
`dragon-cmk`.

Reference docs:

- cert-manager Certificates: <https://cert-manager.io/docs/usage/certificate/>
- ingress-nginx client certificates: <https://kubernetes.github.io/ingress-nginx/examples/auth/client-certs/>
- ingress-nginx backend certificate authentication: <https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#backend-certificate-authentication>
- Traefik ServersTransport: <https://doc.traefik.io/traefik/reference/routing-configuration/kubernetes/crd/http/serverstransport/>
- Traefik TLS options: <https://doc.traefik.io/traefik/reference/routing-configuration/http/tls/tls-options/>

## PostgreSQL included

PostgreSQL is enabled with:

```yaml
postgresql:
  enabled: true
  image:
    repository: dhi.io/postgres
    tag: 18-alpine3.23-fips
```

The database is deployed as a `StatefulSet` and uses a `PersistentVolume` plus a
`PersistentVolumeClaim`. The same persistent volume is mounted for PostgreSQL
data and for the `dragon_cmk` tablespace path used by the SQL scripts.

Override the default development passwords when installing:

```bash
helm install dragon-cmk ./charts \
  --set postgresql.enabled=true \
  --set image.repository=dragon-cmk \
  --set image.tag=latest \
  --set postgresql.adminPassword='<admin-password>' \
  --set postgresql.password='<dragon-cmk-user-password>' \
  --set postgresql.postgresPassword='<postgres-password>'
```

The bundled PostgreSQL container starts with `postgresql.adminUser` (default
`lock_max_user`). The application connects with `postgresql.user` (default
`dragon_cmk_user`), which is created by `storage/squema.sql` during database
initialization.

The application environment includes `VAULT_MODE=local` and
`WORKERS_LIMIT=1000` by default. Override them with chart values when needed:

```bash
helm install dragon-cmk ./charts \
  --set vaultMode=local \
  --set workersLimit=1000
```

TLS and mTLS are enabled by default for REST and gRPC through these
environment values:

```yaml
serverGinTlsEnable: "true"
serverGinMtlsEnable: "true"
serverGrpcTlsEnable: "true"
serverGrpcMtlsEnable: "true"
```

By default, the chart creates a static hostPath `PersistentVolume` at
`/var/lib/dragon-cmk/postgresql`. Change it for the target cluster:

```bash
helm install dragon-cmk ./charts \
  --set postgresql.enabled=true \
  --set postgresql.persistence.persistentVolume.hostPath=/data/dragon-cmk/postgresql
```

## External PostgreSQL

By default, the bundled PostgreSQL is disabled. Set the external connection
values when installing:

```bash
helm install dragon-cmk ./charts \
  --set postgresql.host=postgresql.example.local \
  --set postgresql.adminUser=lock_max_user \
  --set postgresql.adminPassword='<admin-password>' \
  --set postgresql.user=dragon_cmk_user \
  --set postgresql.database=lock_max_db \
  --set postgresql.password='<dragon-cmk-user-password>'
```

## Existing password secret

Create or reuse a secret with the PostgreSQL passwords. `PGPASSWORD` is used by
the application and by the database initialization flow for the
`dragon_cmk_user` password:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: dragon-cmk-postgresql
type: Opaque
stringData:
  PGPASSWORD: '<dragon-cmk-user-password>'
  PGADMIN_PASSWORD: '<admin-password>'
  POSTGRESQL_POSTGRES_PASSWORD: '<postgres-password>'
```

Then install with:

```bash
helm install dragon-cmk ./charts \
  --set postgresql.existingSecret=dragon-cmk-postgresql \
  --set postgresql.existingSecretPasswordKey=PGPASSWORD \
  --set postgresql.existingSecretAdminPasswordKey=PGADMIN_PASSWORD \
  --set postgresql.existingSecretPostgresPasswordKey=POSTGRESQL_POSTGRES_PASSWORD
```

## Resources

The default application deployment runs 2 pods. Each app pod requests `500m`
CPU and `250Mi` memory, with limits of `1000m` CPU and `1Gi` memory.
