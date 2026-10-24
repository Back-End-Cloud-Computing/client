#!/usr/bin/env bash
# Senhas entram pelo ambiente e por arquivos temporários privados, nunca pelo Git.
set -euo pipefail
umask 077
: "${DB_PASSWORD:?Defina DB_PASSWORD no ambiente antes de executar}"
secret_dir="$(mktemp -d)"
trap 'rm -rf "$secret_dir"' EXIT
printf '%s' "$DB_PASSWORD" > "$secret_dir/db-password"
kubectl create secret generic client-credentials \
  --from-file=db-password="$secret_dir/db-password" \
  --dry-run=client -o yaml | kubectl apply -f -
