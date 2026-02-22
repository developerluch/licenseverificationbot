FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# cache-bust: v2
RUN CGO_ENABLED=0 GOOS=linux go build -o /bot .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bot /bot
CMD ["/bot"]
