# Build stage
FROM golang:1.20-alpine3.18 AS builder
WORKDIR /app
COPY . .
RUN go build -o main main.go

# Run stage
FROM alpine3.18
WORKDIR /app
COPY --from=builder /app/main .

# Port should match the value in app.env
EXPOSE 8080
CMD [ "/app/main" ]
