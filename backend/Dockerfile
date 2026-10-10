FROM --platform=$BUILDPLATFORM golang:1.26.8-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /server ./cmd/server
FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata wget && addgroup -g 10001 app && adduser -D -H -u 10001 -G app app && mkdir -p /data/media /data/keys && chown -R app:app /data
COPY --from=build /server /usr/local/bin/server
USER app
ENV LISTEN_ADDR=:8080 DATA_DIR=/data APP_ENV=production
EXPOSE 8080
ENTRYPOINT ["server"]
