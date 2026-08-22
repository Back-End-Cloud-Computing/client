#!/usr/bin/env bash
# Busca a chave publica do Authorization Service.
#
# Este servico so precisa da chave PUBLICA: ela permite verificar a assinatura
# dos tokens, nunca emitir tokens novos. A chave privada fica somente no
# Authorization.
#
# A chave nao entra no Git (ver .gitignore) — cada ambiente busca a sua.
set -euo pipefail

AUTH_URL="${AUTH_URL:-http://localhost:8081}"
DEST="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/keys"
mkdir -p "$DEST"

echo "Buscando a chave publica em $AUTH_URL/auth/public-key ..."

if ! curl -fsS "$AUTH_URL/auth/public-key" -o "$DEST/public.pem"; then
  echo "Nao foi possivel obter a chave. O Authorization Service esta no ar?" >&2
  echo "Suba-o antes (docker compose up no repositorio authorization)." >&2
  exit 1
fi

if ! grep -q "BEGIN PUBLIC KEY" "$DEST/public.pem"; then
  echo "A resposta nao parece uma chave publica em PEM." >&2
  rm -f "$DEST/public.pem"
  exit 1
fi

echo "Chave publica salva em $DEST/public.pem"
