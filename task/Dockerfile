FROM golang:1.17-buster as builder

WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o application cmd/main.go

FROM alpine:3.15.4
WORKDIR /app
COPY --from=builder /app/application /app/application
COPY *.yaml ./
EXPOSE 3000
EXPOSE 4000
EXPOSE 9000
CMD ["/app/application", "-c", "/app/config.yaml"]