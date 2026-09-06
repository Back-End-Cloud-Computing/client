# Client no Kubernetes

Manifests do Lab 05. Sobem o PostgreSQL e três réplicas do Client.

```
Service client :8082
        |
 Service Discovery
        |
 +------+------+------+
 |             |      |
Pod           Pod    Pod
 |             |      |
 +------+------+------+
        |
        | postgres-client:5432
        v
 Service postgres-client
        |
    Pod PostgreSQL
```

## Subir

O Authorization precisa estar no cluster antes, porque o Client valida os tokens
com a chave pública dele.

```bash
./k8s/criar-configmap.sh ../authorization
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/client.yaml
kubectl wait --for=condition=ready pod -l app=client --timeout=180s
```

## ConfigMap, não Secret

O Client recebe apenas a **chave pública**, que permite verificar tokens mas não
emitir. Por isso ela vai num ConfigMap: o tipo do objeto comunica que aquilo não
é segredo. A chave privada fica só no Authorization, num Secret.

## Ver a distribuição entre as réplicas

```bash
kubectl run teste --image=curlimages/curl:latest -it --rm -- sh
```

```sh
for i in 1 2 3 4 5 6 7 8 9 10; do curl -s http://client:8082/instance; echo; done
```

O endpoint `/instance` devolve o nome do Pod que respondeu. Fora do cluster
devolve `local`.

## Comunicação entre os dois serviços

Ainda de dentro do cluster, o mesmo token vale nos dois:

```sh
T=$(curl -s -X POST http://authorization:8081/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"k8s@ganjj.com","password":"senhaSegura123"}' \
  | sed 's/.*"accessToken":"\([^"]*\)".*/\1/')

curl -s -o /dev/null -w "%{http_code}\n" -X POST http://client:8082/clients \
  -H "Authorization: Bearer $T" -H "Content-Type: application/json" \
  -d '{"nome":"Teste","telefone":"(11) 98765-4321","endereco":{"logradouro":"Rua A","numero":"1","cidade":"Sao Paulo","estado":"SP","cep":"01310-100"}}'
```

Nenhum dos dois precisa saber o IP do outro: `authorization` e `client` são
resolvidos pelo DNS interno do cluster.

## Autorrecuperação

```bash
kubectl delete pod <nome-de-um-pod>
kubectl get pods -l app=client
```

O Deployment recria o Pod para manter as três réplicas declaradas.

## Acessar do host

**Via port-forward:**

```bash
kubectl port-forward service/client 8082:8082
```

**Via Ingress** — ver a explicação equivalente em `authorization/k8s/README.md`:

```bash
kubectl apply -f k8s/ingress.yaml
kubectl get svc -n ingress-nginx ingress-nginx-controller   # pega o EXTERNAL-IP
curl -H "Host: client.ganjj.local" http://<EXTERNAL-IP>/health
```

## Sobre a imagem

Publicada em [eduardofabri/client](https://hub.docker.com/r/eduardofabri/client).
Para publicar uma versão nova:

```bash
docker build -t eduardofabri/client:1.2.0 .
docker push eduardofabri/client:1.2.0
```

E atualize a tag em `image:` no `client.yaml`.

## Limitação conhecida

O PostgreSQL roda como Deployment sem volume persistente: se o Pod for recriado,
o banco começa vazio e a aplicação recria a tabela. Um banco de verdade pediria
StatefulSet com PersistentVolumeClaim.
