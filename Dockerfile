FROM golang:1.24-alpine AS build

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY . .

RUN go mod download

ENV CGO_ENABLED=1 CC=gcc

RUN go build -o ./.bin/dv-processing ./cmd/app/

FROM alpine:latest

WORKDIR /app

RUN mkdir -p /app/configs

COPY --from=build /app/.bin/dv-processing /app/

COPY ./artifacts/scripts/start.sh /app/start.sh
RUN chmod +x /app/start.sh

EXPOSE 9000

CMD ["/app/start.sh"]
