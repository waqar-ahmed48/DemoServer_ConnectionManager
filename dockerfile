FROM golang:latest as builder
WORKDIR /DemoServer_ConnectionManager
COPY go.mod go.sum swagger.yaml ./
RUN go mod download
COPY . .
RUN go build -o DemoServer_ConnectionManager .


#FROM gcr.io/distroless/base-debian11
FROM cgr.dev/chainguard/glibc-dynamic
COPY --from=builder /DemoServer_ConnectionManager/DemoServer_ConnectionManager .
COPY --from=builder /DemoServer_ConnectionManager/demoserver_connectionmanager_env_config.yml .

EXPOSE 5678
CMD ["/DemoServer_ConnectionManager", "./demoserver_connectionmanager_env_config.yml"]