# Build go
FROM golang:alpine AS builder
ARG VERSION=dev

WORKDIR /build
COPY . .

RUN go mod tidy
RUN go build -v -o lightsailMon -trimpath -ldflags "-s -w \
    -X 'github.com/Septrum101/lightsailMon/config.date=$(date -Is)' \
    -X 'github.com/Septrum101/lightsailMon/config.version=$VERSION' \
    " ./cmd

# Release
FROM alpine
RUN apk --update --no-cache add tzdata ca-certificates
ENV TZ Asia/Shanghai

WORKDIR /app
COPY --from=builder /build/lightsailMon .
ENTRYPOINT ["/app/lightsailMon"]
