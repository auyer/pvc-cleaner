FROM golang:alpine AS builder

RUN apk add --no-cache ca-certificates git openssh-client && rm -rf /var/cache/apk/*

WORKDIR /app
COPY ./* /app/

RUN CGO_ENABLED=0 GOOS=linux \
    go build --ldflags "-s -w" -a -installsuffix cgo -o main main.go

FROM alpine
RUN apk --no-cache add tzdata

COPY --from=builder /app/main /app/main
WORKDIR /app

CMD ["/app/main", "-c", "config.yaml"]