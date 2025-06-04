# syntax=docker/dockerfile:1
FROM golang:1.24-alpine

WORKDIR /app

# Copiamos el directorio proto primero
COPY . .

# Copiamos los sources del servicio
COPY service-c /app/service-c

# Instalar herramientas necesarias
RUN cd service-c && go mod tidy && go build -o main .

EXPOSE 8088 50051 50052

CMD ["./service-c/main"]
