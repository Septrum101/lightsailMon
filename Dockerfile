# Build go
FROM golang:alpine AS builder
WORKDIR /app
COPY . .
ENV CGO_ENABLED=0
RUN go mod download
RUN go build -v -o lsMon -trimpath -ldflags "-s -w" ./cmd

# Release
FROM alpine
RUN apk --update --no-cache add tzdata ca-certificates
ENV TZ Asia/Shanghai

RUN mkdir /etc/LightsailMon/
COPY --from=builder /app/lsMon /usr/local/bin

ENTRYPOINT ["lsMon"]
