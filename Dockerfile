FROM golang:1.17 as build

ADD . /app
WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux go build

FROM alpine:3.15
COPY --from=build /app/ShadowTest /usr/bin/

ENTRYPOINT /usr/bin/ShadowTest
