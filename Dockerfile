FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY app.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /otel-go-statsd-demo .

FROM alpine:latest

WORKDIR /

COPY --from=build /otel-go-statsd-demo /otel-go-statsd-demo

EXPOSE 6767

CMD ["/otel-go-statsd-demo"]
