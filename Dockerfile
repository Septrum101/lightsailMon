# Build go
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY . .
ENV CGO_ENABLED=0
RUN go mod download
RUN go build -v -o lsMon -trimpath -ldflags "-s -w -buildid=" ./cmd

# Release
FROM alpine
RUN apk --update --no-cache add tzdata ca-certificates \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN mkdir /etc/LightsailMon/
COPY --from=builder /app/lsMon /usr/local/bin

ENTRYPOINT ["lsMon"]
