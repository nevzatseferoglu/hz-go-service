# syntax=docker/dockerfile:1

FROM golang:1.17-alpine AS build

WORKDIR /app

COPY . .
RUN go mod download

RUN go build -o sample-application

FROM alpine

COPY --from=build /app/sample-application /app/binary

CMD ["/app/binary"]