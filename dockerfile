FROM golang:latest as builder
WORKDIR /DemoServer_ConnectionManager
COPY go.mod go.sum swagger.yaml ./
RUN go mod download
COPY . .
RUN go build -o main .


#FROM gcr.io/distroless/base-debian11
FROM cgr.dev/chainguard/glibc-dynamic
COPY --from=builder /DemoServer_ConnectionManager/main .
COPY --from=builder /DemoServer_ConnectionManager/demoserver_connectionmanager_env_config.yml .

EXPOSE 5678
CMD ["/main", "./demoserver_connectionmanager_env_config.yml"]