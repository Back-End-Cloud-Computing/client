# Client Service — GANJJ

Microsserviço de perfil de cliente do e-commerce **GANJJ**. Guarda dados de cadastro
(nome, telefone, endereço) — nenhuma credencial: e-mail e senha são do
[authorization](https://github.com/Back-End-Cloud-Computing/authorization).

## Stack

- **Go** (roteamento nativo do `net/http`, sem framework)
- **PostgreSQL** (driver `pgx`)

## Endpoints

| Método | Rota | Autenticação | Descrição |
|---|---|---|---|
| `POST` | `/clients` | Bearer | Cria o perfil da conta autenticada |
| `GET` | `/clients/me` | Bearer | Perfil da conta autenticada |
| `PUT` | `/clients/me` | Bearer | Atualiza o próprio perfil |
| `DELETE` | `/clients/me` | Bearer | Remove o próprio perfil |
| `GET` | `/clients` | Bearer + ADMIN | Lista todos os perfis |
| `GET` | `/clients/{id}` | Bearer + ADMIN | Busca um perfil por id |
| `GET` | `/health` | pública | Usada pelo healthcheck |

A conta sempre vem da claim `sub` do token, **nunca do corpo da requisição** — assim
ninguém consegue criar ou alterar o perfil de outra pessoa.

### Exemplo de corpo

```json
{
  "nome": "Eduardo Fabri",
  "telefone": "(11) 98765-4321",
  "endereco": {
    "logradouro": "Rua das Flores",
    "numero": "123",
    "complemento": "apto 45",
    "bairro": "Centro",
    "cidade": "São Paulo",
    "estado": "SP",
    "cep": "01310-100"
  }
}
```

## Como a autenticação funciona aqui

Este serviço **não chama o Authorization** para autenticar requisições. Ele carrega a
chave pública uma vez, na inicialização, e verifica a assinatura dos tokens localmente.
A chave pública só permite verificar — emitir tokens continua sendo exclusividade do
Authorization, que é o único a ter a chave privada.

A verificação exige, além da assinatura: algoritmo `RS256`, emissor
`ganjj-authorization`, `typ` igual a `access` e um `exp` no futuro. Fixar o algoritmo é
o que impede um token forjado com HMAC usando a chave pública (que é conhecida) como
se fosse senha — há teste cobrindo esse caso.

## Divisão com o serviço `authorization`

| | [authorization](https://github.com/Back-End-Cloud-Computing/authorization) | client |
|---|---|---|
| Guarda | e-mail, senha (BCrypt), papel | nome, telefone, endereço |
| Banco | Oracle | PostgreSQL |
| Linguagem | Java | Go |

Os dois se relacionam pelo **mesmo id de conta** e não se chamam entre si.

## Rodando com Docker

O Authorization precisa estar no ar primeiro, para fornecer a chave pública:

```bash
docker network create ganjj-net
./scripts/fetch-public-key.sh
docker compose up --build
```

O serviço sobe em `http://localhost:8082`.

### Fluxo completo

```bash
TOKEN=$(curl -s -X POST http://localhost:8081/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"cliente@ganjj.com","password":"senhaSegura123"}' \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['accessToken'])")
```

```bash
curl -X POST http://localhost:8082/clients \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"nome":"Eduardo Fabri","telefone":"(11) 98765-4321","endereco":{"logradouro":"Rua das Flores","numero":"123","bairro":"Centro","cidade":"São Paulo","estado":"SP","cep":"01310-100"}}'
```

```bash
curl http://localhost:8082/clients/me -H "Authorization: Bearer $TOKEN"
```

## Rodando local, sem Docker

Suba um PostgreSQL, pegue a chave pública e execute:

```bash
go run ./cmd/api
```

## Testes

```bash
go test ./...
```

Os testes usam um repositório em memória e chaves geradas na hora, então não precisam
de banco nem do Authorization no ar.

## Configuração

Tudo por variável de ambiente — o mesmo binário roda em desenvolvimento, teste e
produção trocando só os valores.

| Variável | Padrão | Descrição |
|---|---|---|
| `SERVER_PORT` | `8082` | Porta HTTP |
| `DB_HOST` | `localhost` | Host do PostgreSQL |
| `DB_PORT` | `5432` | Porta do PostgreSQL |
| `DB_NAME` | `ganjj_client` | Nome do banco |
| `DB_USER` | `ganjj_client` | Usuário do banco |
| `DB_PASSWORD` | `ganjj_client` | Senha do banco |
| `DB_SSLMODE` | `disable` | Modo SSL da conexão |
| `JWT_PUBLIC_KEY_PATH` | `keys/public.pem` | Chave pública do Authorization |
| `JWT_ISSUER` | `ganjj-authorization` | Emissor exigido no token |
| `CORS_ALLOWED_ORIGINS` | portas de dev | Origens do frontend, separadas por vírgula |

A chave em `keys/` não é versionada: cada ambiente busca a sua com
`./scripts/fetch-public-key.sh`.

## Time

Parte do projeto GANJJ (5 microsserviços poliglotas). Mantido por Eduardo Fabri, que
também mantém o serviço `authorization`.
