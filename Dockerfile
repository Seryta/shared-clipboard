FROM golang:1.20.4 as builder

WORKDIR /shared-clipboard

COPY go.mod go.mod
COPY go.sum go.sum

RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod download

COPY main.go main.go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags -a -o shared-clipboard-server main.go

FROM alpine
WORKDIR /shared-clipboard
COPY --from=builder /shared-clipboard/shared-clipboard-server /shared-clipboard/shared-clipboard-server
COPY web /shared-clipboard/web

EXPOSE 8080

ENTRYPOINT ["/shared-clipboard/shared-clipboard-server"]