# Build go
FROM golang:alpine AS builder

ARG VERSION

WORKDIR /app
COPY . .

RUN go mod tidy
RUN go build -v -o lightsailMon -trimpath -ldflags "-s -w \
    -X 'github.com/Septrum101/lightsailMon/config.date=$(date -Iseconds)' \
    -X 'github.com/Septrum101/lightsailMon/config.version=$VERSION' \
    " ./cmd

# Release
FROM alpine
RUN apk --update --no-cache add tzdata ca-certificates
ENV TZ Asia/Shanghai

COPY --from=builder /app/lightsailMon /app/lightsailMon
ENTRYPOINT ["/app/lightsailMon"]
