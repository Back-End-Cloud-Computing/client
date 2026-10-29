# Entrega local — client

Atualizado em 29/10/2026. Este guia pertence ao serviço client; detalhes da API e configuração completa estão no [README](../README.md).

## Desenvolvimento local e apresentação N2

Este guia descreve o laboratório de desenvolvimento e validação local usado nesta etapa. A decisão temporária de trabalhar localmente não define o destino final da apresentação. A implantação compartilhada da N2 segue o [kit Amazon EKS do api-gateway](https://github.com/Back-End-Cloud-Computing/api-gateway/tree/main/eks), coordenado pelo grupo. O Gateway/Ingress é a entrada externa única; client, authorization e support continuam serviços internos, sem necessidade de uma URL pública por serviço.

O kit aplica os manifests no namespace do contexto kubectl, default no fluxo padrão. DNS curto como authorization:8081 funciona para consumidores no mesmo namespace; authorization.default.svc:8081 identifica explicitamente o namespace default. O DNS com ganjj-n2 só vale no laboratório onde esse namespace foi criado/aplicado. Esta revisão é documental: não cria nem modifica infraestrutura AWS.

## Responsabilidade e dependências

O client mantém nome, telefone e endereço no PostgreSQL. Não armazena senha nem emite tokens. O identificador da conta vem da claim sub do access token emitido pelo authorization.

Requisitos: Go na versão de go.mod (1.27.0), PostgreSQL e chave pública PEM do mesmo authorization que emitiu o token. Para Docker Compose, usar a rede ganjj-net compartilhada e iniciar authorization primeiro.

## Endereços

| Ambiente | Endereço |
|---|---|
| Aplicação no computador | http://localhost:8082 |
| Swagger local | http://localhost:8082/swagger/index.html |
| Health local | http://localhost:8082/health |
| Réplica que respondeu | http://localhost:8082/instance |
| Métricas | /metrics, coletadas internamente no Kubernetes |

Não há hospedagem pública nova. localhost aponta para o próprio computador/container; DNS .svc só funciona no cluster correspondente. A imagem publicada eduardofabri/client:1.2.0 não cria um endpoint público por si só.

## Iniciar

No repositório authorization, iniciar o serviço com sua chave persistente e banco. Depois, neste repositório:

```bash
./scripts/fetch-public-key.sh
docker compose up --build
```

A rede ganjj-net precisa existir (docker network create ganjj-net quando ainda não existir). Para buscar a chave de outra porta/host acessível, definir AUTH_URL antes do script. Se PostgreSQL já ocupar a porta externa 5432, usar DB_HOST_PORT=5433 no Compose.

Sem Docker, configurar PostgreSQL pelas variáveis DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD e DB_SSLMODE, obter a chave e executar go run ./cmd/api. O Go lê variáveis do processo, não importa um arquivo .env automaticamente. A tabela completa está no README.

## Integrar e conferir

1. Registrar/autenticar uma conta no authorization e obter accessToken.
2. Criar perfil com POST /clients, usando Authorization: Bearer e o payload do README.
3. Consultar/atualizar/remover o próprio perfil por GET/PUT/DELETE /clients/me.
4. Para GET /clients e GET /clients/{id}, usar conta ADMIN. CLIENTE deve receber 403 nessas rotas.

O client verifica assinatura RS256, issuer ganjj-authorization, typ=access e expiração localmente. O refresh token não abre as rotas protegidas. A chave pública é carregada ao iniciar: após trocar a chave do authorization, atualizar o PEM e reiniciar o client. Nunca fornecer chave privada ao client.

## Testes e evidências

```bash
go test ./... -short
go test -race ./...
go vet ./...
go build ./cmd/api
```

O modo short não executa a integração PostgreSQL. A suíte completa usa PostgreSQL real via Testcontainers e exige Docker. CI também verifica formatação e constrói a imagem.

Validações anteriores registradas em 25/10: perfil com JWT real, restrições CLIENTE/ADMIN, rejeição de refresh/assinatura inválida, três réplicas, Helm, Prometheus/Grafana e imagem pública multiarch. Essas execuções não foram repetidas nesta revisão documental.

- [Integração JWT/perfis](evidence/2026-10-25/auth-client.json)
- [Imagem publicada](evidence/2026-10-25/client-image.json)
- [Cluster](evidence/2026-10-25/cluster.json) e [Ingress](evidence/2026-10-25/ingress.json)
- [Dashboard Grafana](../k8s/observability/README.md)
- [Instalar por Helm](../helm/client/README.md) e [Secrets Kubernetes](../k8s/README.md)

## Pendências

O ensaio real do grupo (client #7) continua pendente e por último. A revisão documental foi enviada à branch própria codex/documentacao-client; a integração na main depende de revisão. A etapa atual de desenvolvimento permanece local; a apresentação compartilhada segue o kit EKS do grupo. Não publicar senhas de banco ou tokens em documentação/issues.

Matriz completa de aceite das issues deste serviço: [aceite-final.md](aceite-final.md).
