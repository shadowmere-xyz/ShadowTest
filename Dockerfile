FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod go.sum /app/
RUN go mod download

COPY . /app

ARG VERSION
ARG COMMIT
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.Version=$VERSION -X main.GitCommit=$COMMIT"

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /app/ShadowTest /usr/bin/
EXPOSE 8080

ENTRYPOINT ["/usr/bin/ShadowTest"]
