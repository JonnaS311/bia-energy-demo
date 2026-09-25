# Etapa 1: build estático (spec backend §9)
FROM golang:1.22-alpine AS build
WORKDIR /src
ENV GOTOOLCHAIN=local CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o /out/api ./cmd/api

# Etapa 2: runtime mínimo con certificados para HTTPS hacia DeepSeek
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -H -u 10001 app
COPY --from=build /out/api /usr/local/bin/api
USER app
ENV PORT=8080 DATA_DIR=/app/data
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --start-period=10s --retries=12 CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1
ENTRYPOINT ["/usr/local/bin/api"]
