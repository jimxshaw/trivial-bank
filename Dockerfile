FROM golang:1.20-alpine3.18
WORKDIR /app
COPY . .
RUN go build -o main main.go

# Port should match the value in app.env
EXPOSE 8080
CMD [ "/app/main" ]
