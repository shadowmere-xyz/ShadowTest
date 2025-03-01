FROM golang:1.24 AS build

WORKDIR /app

COPY go.mod go.sum /app/
RUN go mod download

COPY . /app

RUN CGO_ENABLED=0 GOOS=linux go build

FROM alpine:3.21
COPY --from=build /app/ShadowTest /usr/bin/

ENV APP_USER=shadowtest
ENV APP_GROUP=shadowtestgroup
RUN addgroup -S "$APP_GROUP" && adduser -H -S "$APP_USER" -G "$APP_GROUP" -s /sbin/nologin \
    && chown -R "$APP_USER":"$APP_GROUP" /usr/bin/ShadowTest
USER $APP_USER

EXPOSE 8080
HEALTHCHECK --start-period=3s --interval=5s --timeout=1s --retries=3 CMD ["wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]

ENTRYPOINT ["/usr/bin/ShadowTest"]
