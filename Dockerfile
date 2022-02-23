FROM golang:1.17-alpine3.15 as builder
ENV USER=appuser
ENV UID=10001
RUN apk update && apk upgrade && apk add --no-cache git ca-certificates tzdata && update-ca-certificates
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"
WORKDIR /app
COPY . .
RUN go mod download
RUN go mod verify
RUN CGO_ENABLED=0 \
    GOOS=linux GOARCH=amd64 \
    go build -ldflags '-w -s -extldflags "-static"' -a \
    -o application hello/main.go

FROM scratch
ENV USER=appuser
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /app/application /application
COPY --from=builder /app/wallet /wallet
USER appuser:appuser
CMD ["/application"]