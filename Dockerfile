FROM golang:1.24 as builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=1 go build -o /parser main.go

FROM alpine:latests
RUN apk add --no-cache ca-certificates
COPY --from=builder /parser /parser
CMD ["/parser"]