# syntax=docker/dockerfile:1
FROM golang:1.24-alpine

WORKDIR /app

# Copiamos el directorio proto primero
COPY . .

# Copiamos los sources del servicio
COPY service-b /app/service-b

# Instalar herramientas necesarias
RUN cd service-b && go mod tidy && go build -o main .

EXPOSE 8080 50051 50052

CMD ["./service-b/main"]
