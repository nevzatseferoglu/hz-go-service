# syntax=docker/dockerfile:1

FROM golang:1.17-alpine AS build

WORKDIR /app

COPY . .
RUN go mod download

RUN go build -o /sample-application

FROM scratch

COPY --from=build /sample-application /sample-application

ENTRYPOINT ["/sample-application"]