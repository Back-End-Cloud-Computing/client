# Roteiro de apresentação N2 — Client

Responsável: Eduardo. Apresentação: **10/11/2026**. Ensaio com o grupo até **07/11**, ligado à issue #7.

Este material prepara a fala e a demonstração; não registra que o ensaio aconteceu. Usar o EKS compartilhado na demonstração do grupo e o ambiente local para treinar passos.

## 1. O que falar e em quanto tempo

Sugestão: **3 minutos**, não um tempo medido.

| Janela planejada | Fala sugerida | O que mostrar |
|---|---|---|
| 0:00–0:30 | “O Client guarda nome, telefone e endereço. Ele é Go com PostgreSQL; a senha e a identidade ficam no Authorization em Oracle.” | Responsabilidade clara, duas linguagens e bancos separados |
| 0:30–1:00 | “O cliente entra pelo Gateway e o serviço conversa com o seu PostgreSQL por Service Discovery, sem usar IP de Pod.” | Deployments/Services e réplica atual |
| 1:00–2:10 | “Vou criar e consultar o perfil da conta autenticada. O dono vem do sub do JWT, e não de um account_id enviado no JSON.” | POST /clients e GET /clients/me |
| 2:10–2:40 | “Sem token o acesso é negado. Uma conta CLIENTE não pode listar perfis como ADMIN.” | 401 sem token e 403 na listagem |
| 2:40–3:00 | “A validação da assinatura usa a chave pública carregada; não faço uma chamada ao Authorization em cada requisição.” | Resumo e passagem para o fluxo do pedido |

## 2. Preparação

Antes da tela compartilhada, preparar uma conta CLIENTE **fictícia e descartável de ensaio** e um access token do Authorization do mesmo cluster. O [roteiro do Authorization](https://github.com/Back-End-Cloud-Computing/authorization/blob/codex/preparacao-ensaio/docs/apresentacao-n2.md) mostra como carregar o token sem exibi-lo. A variável GANJJ_TOKEN_CLIENTE deve existir neste Terminal.

Os comandos desta seção usam os nomes dos manifests k8s/ do kit EKS; o chart do laboratório local pode usar nomes diferentes. Confirmar o contexto e ajustar o namespace antes de executar. Variáveis exportadas valem somente na aba de Terminal em que foram definidas.

```bash
export GANJJ_NAMESPACE=default
export GANJJ_BASE_URL="http://ELB_DO_GATEWAY"
kubectl config current-context
kubectl get deployment client -n "$GANJJ_NAMESPACE"
kubectl get service client postgres-client -n "$GANJJ_NAMESPACE"
```

Trocar o ELB e confirmar o contexto com o responsável pelo EKS. O kit pode reduzir réplicas: mostrar o número real, sem prometer três réplicas de Client no cluster compartilhado. A evidência de balanceamento da solução deve usar um serviço que esteja replicado no kit, combinado com os responsáveis por Gateway/Promotion. Um port-forward escolhe um Pod e não comprova balanceamento pelo Ingress.

## 3. Criar e consultar o perfil

Usar somente dados fictícios, sem conta de cliente real. Esperado no primeiro cadastro: **201**. Este telefone é dado sintético de validação de perfil, não destinatário de envio.

```bash
curl -i -X POST "$GANJJ_BASE_URL/clients" \
  -H "Authorization: Bearer $GANJJ_TOKEN_CLIENTE" \
  -H 'Content-Type: application/json' \
  -d '{"nome":"Cliente Ensaio","telefone":"(11) 90000-0000","endereco":{"logradouro":"Rua Exemplo","numero":"100","bairro":"Centro","cidade":"Curitiba","estado":"PR","cep":"80000000"}}'
curl -i -H "Authorization: Bearer $GANJJ_TOKEN_CLIENTE" "$GANJJ_BASE_URL/clients/me"
```

GET esperado: **200**, com os dados do perfil. Se o POST retornar **409**, essa conta já tem perfil; consultar o existente ou preparar outra conta de ensaio antes da apresentação. Um GET **404** com token válido indica perfil ainda não criado.

### Acesso protegido

```bash
curl -sS -o /dev/null -w 'HTTP %{http_code}\n' "$GANJJ_BASE_URL/clients/me"
curl -sS -o /dev/null -w 'HTTP %{http_code}\n' -H "Authorization: Bearer $GANJJ_TOKEN_CLIENTE" "$GANJJ_BASE_URL/clients"
```

Resultados esperados: **401** e **403**, respectivamente. A listagem exige ADMIN no próprio Client. Não exibir um token inteiro na tela para explicar o JWT.

## 4. Perguntas para treinar

| Pergunta | Resposta curta |
|---|---|
| Por que Go? | Serviço de perfil com net/http e pgx; contribui para a diversidade de linguagem do projeto. |
| Por que PostgreSQL? | Dados de perfil estruturados e persistência relacional própria do serviço. |
| Por que não usar o banco do Authorization? | Database per Service: evita compartilhar tabelas e responsabilidades. |
| Como uma pessoa não altera o perfil de outra? | A conta vem da claim sub validada; o corpo não escolhe o dono. |
| O Client consulta Authorization a cada chamada? | Não; valida RS256, issuer, tipo access e expiração com a chave pública carregada. |
| O que acontece se a chave mudar? | Atualizar o PEM desse ambiente e reiniciar o consumidor; tokens/chaves de clusters diferentes não são intercambiáveis. |

## 5. Extras e plano B

Se houver tempo, mostrar requisições/erros/latência no Grafana já preparado. EKS/REST/Service Discovery e responsabilidade do serviço são o foco do roteiro consolidado; Grafana não deve consumir o tempo do fluxo principal.

- [JWT e perfis no laboratório](evidence/2026-10-25/auth-client.json).
- [Cluster histórico](evidence/2026-10-25/cluster.json).
- [Dashboard histórico, extra](evidence/2026-10-25/grafana-client.jpg).
- [Guia operacional](entrega-local.md) e [aceite](aceite-final.md).

Se usar captura ou Minikube como plano B, identificar como evidência histórica/local. Não declarar que o EKS foi demonstrado ao vivo se não foi.

## 6. Checklist e registro

- [ ] Confirmar contexto EKS, namespace, ELB e Services.
- [ ] Preparar conta/token de ensaio e conferir se já existe perfil.
- [ ] Ensaiar POST/GET e as respostas 401/403.
- [ ] Combinar a passagem de fala para Cart/Order.
- [ ] Cronometrar com o grupo e preencher [registro-ensaio.md](registro-ensaio.md).

A issue #7 continua aberta até existir ensaio humano e tempos reais anotados.

## Modelos usados

A estrutura segue os roteiros dos colegas: demonstração com comando/resultado, decisões para defender, perguntas e checklist.

- [Runbook do Rodrigo — Order](https://github.com/Back-End-Cloud-Computing/order/blob/main/docs/PRESENTATION-RUNBOOK.md).
- [Roteiro do Promotion](https://github.com/Back-End-Cloud-Computing/promotion/blob/main/docs/apresentacao-n2.md).
- [Rubrica atual transcrita pelo grupo](https://github.com/Back-End-Cloud-Computing/promotion/blob/main/docs/rubrica-n2.md). O documento registra a fonte Canvas e a dúvida sobre a soma dos pontos; este guia não redefine a avaliação.
- [Kit compartilhado EKS](https://github.com/Back-End-Cloud-Computing/api-gateway/tree/main/eks).

Preparação escrita em 29/10/2026. Tempos abaixo são sugestão; tempos reais só entram no [registro de ensaio](registro-ensaio.md) depois da atividade com o grupo.
