# Validação N2 (client)

Preparado em 24/10/2026 para revisão. Não representa conclusão das etapas manuais.

- CI, `/metrics`, credenciais em Secret, chart Helm e dashboard JSON implementados.
- Validação local: `go test -race ./...`, `go vet ./...` e build Docker; `helm lint` e renderização dos templates.
- Publicação Docker Hub pendente: configurar `DOCKERHUB_USERNAME` e `DOCKERHUB_TOKEN`, executar o workflow de release para `1.2.0`, conferir `docker buildx imagetools inspect eduardofabri/client:1.2.0` e confirmar acesso público.
- Cluster pendente: o ambiente não tinha contexto Kubernetes configurado. Criar os Secrets/chaves e validar `helm install` num Minikube limpo; gerar tráfego e capturar o dashboard.
- Conferir os conteúdos das aulas de Helm (27/10) e observabilidade (03/11) antes do aceite final.
- Ensaio com o grupo pendente, com tempo por pessoa anotado até 07/11. Apresentação em 10/11.

## Roteiro para o ensaio

1. Explicar por que Go com roteador nativo e PostgreSQL para o perfil relacional.
2. Explicar JWT RS256: quem guarda a privada, como a pública é distribuída e por que múltiplas réplicas validam o mesmo token.
3. Demonstrar login, requisição protegida, 401 sem token e 403 sem papel adequado.
4. Mostrar Deployments, Services, réplicas, probes, limites e credenciais por Secret.
5. Gerar tráfego; mostrar requisições, erros e latência no Grafana.
6. Anotar duração real e ajustar o roteiro com o grupo.
