# Build stage
FROM golang:1.20.2-alpine3.17 as builder
WORKDIR /app
COPY . .
RUN go build -o build/main main.go

FROM alpine:3.17
WORKDIR /app
COPY --from=builder /app/build/main ./build

EXPOSE 8080

CMD [ "build/main" ]