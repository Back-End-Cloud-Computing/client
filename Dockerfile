# Etapa 1 — compila o binario dentro da propria imagem, para o build nao
# depender do que esta instalado na maquina de quem roda.
FROM golang:1.27 AS build
WORKDIR /build

# As dependencias sao baixadas antes do codigo: enquanto go.mod/go.sum nao
# mudarem, o Docker reaproveita esta camada.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO desligado gera um binario estatico, que roda numa imagem final sem
# bibliotecas do sistema.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/api ./cmd/api

# Etapa 2 — imagem final: so o binario e os certificados raiz.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=build /build/api /app/api

EXPOSE 8082
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]
