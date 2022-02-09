FROM golang:1.17-alpine3.15 as builder

# Copy local code to the container image.
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build hello/main.go -v -o application

# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
FROM alpine:3.15.0
RUN apk add --no-cache ca-certificates sed
COPY  --from=builder /app/application /application
CMD ["/application"]