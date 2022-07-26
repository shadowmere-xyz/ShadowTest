FROM golang:1.18 as build

ADD . /app
WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux go build

FROM alpine:3.16
COPY --from=build /app/ShadowTest /usr/bin/


EXPOSE 8080
HEALTHCHECK CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT /usr/bin/ShadowTest
