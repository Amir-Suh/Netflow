# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/netflow-api ./cmd/api

FROM alpine:3.21
RUN adduser -D -H -u 10001 netflow
USER netflow
WORKDIR /app
COPY --from=build /out/netflow-api /app/netflow-api
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=10 CMD wget -qO- http://localhost:8080/readyz >/dev/null || exit 1
ENTRYPOINT ["/app/netflow-api"]
