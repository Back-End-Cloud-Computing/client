#!/usr/bin/env bash
# Cria o ConfigMap com a chave PUBLICA do Authorization.
#
# E ConfigMap e nao Secret de proposito: esta chave permite apenas verificar
# tokens, nunca emitir. A privada fica somente no Authorization, num Secret.
#
# Uso:
#   ./k8s/criar-configmap.sh                     usa keys/public.pem
#   ./k8s/criar-configmap.sh ../authorization    busca no repo do authorization
set -euo pipefail

RAIZ="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ORIGEM="${1:-$RAIZ}"
CHAVE="$ORIGEM/keys/public.pem"

if [[ ! -f "$CHAVE" ]]; then
  echo "Chave publica nao encontrada em $CHAVE."
  echo "Pegue a do Authorization: ./scripts/fetch-public-key.sh"
  echo "Ou aponte para o repo dele: ./k8s/criar-configmap.sh ../authorization"
  exit 1
fi

kubectl create configmap jwt-public-key \
  --from-file=public.pem="$CHAVE" \
  --dry-run=client -o yaml | kubectl apply -f -

echo
echo "ConfigMap 'jwt-public-key' pronto. Confira com:"
echo "  kubectl describe configmap jwt-public-key"
