# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/middleware .

FROM alpine:3.22
# ca-certificates: outbound HTTPS to Yahoo Finance and logo.dev.
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 10001 app
COPY --from=build /out/middleware /usr/local/bin/middleware

USER app
EXPOSE 3000
ENTRYPOINT ["/usr/local/bin/middleware"]
