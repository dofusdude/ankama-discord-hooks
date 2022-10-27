# syntax=docker/dockerfile:1

FROM golang:1.19-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /ankama-discord-hooks

## Deploy
FROM gcr.io/distroless/static-debian11

LABEL maintainer="stelzo"
USER nonroot:nonroot

COPY --from=build --chown=nonroot:nonroot /ankama-discord-hooks /ankama-discord-hooks

WORKDIR /

EXPOSE 3000

CMD [ "/ankama-discord-hooks" ]