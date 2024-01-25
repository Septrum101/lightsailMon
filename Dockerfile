# Build go
FROM golang:alpine AS builder

WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -v -o lightsailMon -trimpath -ldflags "-s -w \
    -X 'github.com/thank243/lightsailMon/config.date=$(date -Iseconds)'" ./cmd

# Release
FROM alpine
RUN apk --update --no-cache add tzdata ca-certificates
ENV TZ Asia/Shanghai

COPY --from=builder /app/lightsailMon /app/lightsailMon
ENTRYPOINT ["/app/lightsailMon"]
