# syntax=docker/dockerfile:1

FROM golang:1.16 AS builder

WORKDIR /app

COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -tags sample-application -o sample-application -ldflags '-w' .

FROM scratch
COPY --from=builder /app/sample-application /sample-application
ENTRYPOINT ["/sample-application"]
