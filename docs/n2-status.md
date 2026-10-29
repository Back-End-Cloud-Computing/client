# Validação N2 — client

Atualizado em 29/10/2026; as validações técnicas abaixo foram registradas em 25/10/2026. Evidências sanitizadas em `docs/evidence/2026-10-25/`.

## Executado

- Build, análise e testes locais; CI do PR anterior verde (nova revisão será validada novamente).
- Imagem pública Docker Hub `eduardofabri/client:1.2.0`, com linux/amd64 e linux/arm64. Manifest inspecionado e acesso anônimo validado.
- Instalação real por Helm em Minikube limpo (perfil `ganjj-n2`, Kubernetes 1.37.0), credenciais externas ao Git.
- Prometheus coletando o serviço e dashboard Grafana importado por provisionamento. Captura versionada.
- Três réplicas saudáveis; criar/consultar perfil com JWT real; 401 sem token, 403 em rota administrativa para CLIENTE, 200 para ADMIN, 401 para refresh e assinatura adulterada.
- Ingress aceita `/instance` e não encaminha `/metrics` (404).

## Pendências humanas / externas

- Ensaio da apresentação com o grupo, com tempo por pessoa; roteiro pronto, mas não foi declarado um ensaio humano que não ocorreu.

Guia operacional próprio: [entrega-local.md](entrega-local.md). O ambiente permanecerá local; a revisão documental não reexecutou a suíte técnica.
