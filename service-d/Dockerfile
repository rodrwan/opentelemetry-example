# syntax=docker/dockerfile:1
FROM golang:1.24-alpine

WORKDIR /app

# Copiamos el directorio proto primero
COPY . .

# Copiamos los sources del servicio
COPY service-d /app/service-d

# Instalar herramientas necesarias
RUN cd service-d && go mod tidy && go build -o main .

EXPOSE 8089

CMD ["./service-d/main"]
