FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o vllm_serv

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/vllm_serv .
COPY response.json .
COPY stream_res.json .

EXPOSE 3000
ENTRYPOINT ["./vllm_serv"]