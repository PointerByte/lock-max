ARG GO_VERSION=1.25.0
FROM --platform=$TARGETPLATFORM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

RUN --mount=type=cache,target=/var/cache/apk \
    apk --update add \
        build-base

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=1 go build -o /bin/server .

ARG ALPINE_VERSION=3.22.4
FROM alpine:${ALPINE_VERSION} AS final

RUN --mount=type=cache,target=/var/cache/apk \
    apk --update add \
        ca-certificates \
        tzdata \
        && \
        update-ca-certificates

ARG UID=10001
WORKDIR /app
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    appuser

RUN mkdir -p /app/logs && chown -R appuser:appuser /app

COPY --from=build /bin/server /bin/server
COPY application.yaml /app/application.yaml
COPY certs /app/certs

USER appuser

EXPOSE 8080 50051

ENTRYPOINT [ "/bin/server" ]
