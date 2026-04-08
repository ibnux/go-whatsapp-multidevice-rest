# Script para compilar o executável Go na raiz do projeto via Docker
$ErrorActionPreference = 'Stop'

Write-Host "Compilando aplicação Go via Docker..."

docker run --rm -v "${PWD}:/src" -w /src `
    -e GOOS=windows -e GOARCH=amd64 -e CGO_ENABLED=0 `
    golang:1.25-alpine `
    go build -ldflags="-s -w" -a -o dimaskido.exe cmd/main/main.go

if ($LASTEXITCODE -eq 0) {
    Write-Host "dimaskido.exe gerado com sucesso na raiz do projeto."
} else {
    Write-Host "Erro ao compilar o executável."
    exit $LASTEXITCODE
}
