# Busca a chave publica do Authorization Service.
#
# Este servico so precisa da chave PUBLICA: ela permite verificar a assinatura
# dos tokens, nunca emitir tokens novos. A chave privada fica somente no
# Authorization.
#
# A chave nao entra no Git (ver .gitignore) - cada ambiente busca a sua.
#
# Versao PowerShell do fetch-public-key.sh, para quem usa Windows.
# Uso:  .\scripts\fetch-public-key.ps1
#       $env:AUTH_URL = 'http://outro-host:8081'; .\scripts\fetch-public-key.ps1

$ErrorActionPreference = 'Stop'

$authUrl = if ($env:AUTH_URL) { $env:AUTH_URL } else { 'http://localhost:8081' }

$dest = Join-Path (Split-Path -Parent $PSScriptRoot) 'keys'
New-Item -ItemType Directory -Force -Path $dest | Out-Null

$destino = Join-Path $dest 'public.pem'

Write-Host "Buscando a chave publica em $authUrl/auth/public-key ..."

try {
    $resposta = Invoke-WebRequest -Uri "$authUrl/auth/public-key" -UseBasicParsing
}
catch {
    Write-Error @"
Nao foi possivel obter a chave. O Authorization Service esta no ar?
Suba-o antes (docker compose up no repositorio authorization).
"@
    exit 1
}

$conteudo = $resposta.Content

if ($conteudo -notmatch 'BEGIN PUBLIC KEY') {
    Write-Error 'A resposta nao parece uma chave publica em PEM.'
    exit 1
}

# Sem BOM: o Go le o arquivo como PEM puro, e um BOM no inicio quebraria a
# leitura. Set-Content do PowerShell 5.1 escreveria BOM em UTF8.
[System.IO.File]::WriteAllText($destino, $conteudo, (New-Object System.Text.UTF8Encoding $false))

Write-Host "Chave publica salva em $destino"
