# Client Service — GANJJ

Microsserviço de perfil de cliente do e-commerce **GANJJ**. Guarda dados de cadastro
(nome, endereço, contato) — não guarda credencial nenhuma, isso é responsabilidade do
[authorization](https://github.com/Back-End-Cloud-Computing/authorization).

## Stack

- **Go**
- **PostgreSQL** (driver `pgx`)

## Escopo

- CRUD de perfil de cliente
- Referencia o usuário pelo mesmo ID emitido pelo `authorization` no cadastro/login —
  sem duplicar senha ou dado de login aqui
- Valida requisições autenticadas localmente: recebe a chave pública do `authorization`
  (variável de ambiente) e verifica a assinatura do JWT (RS256), sem chamar o
  `authorization` a cada requisição

## Time

Parte do projeto GANJJ (5 microsserviços poliglotas). Mantido por Eduardo Fabri, que
também mantém o serviço `authorization`.
