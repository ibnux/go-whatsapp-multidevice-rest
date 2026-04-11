# Script PowerShell para build e deploy automatizado no registry privado

$RegistryUrl = "registry.hom.lan"
$ImageName = "whatsapp-dimas"

$LatestImageName = "$RegistryUrl/${ImageName}:latest"

Write-Host "[DEPLOY] Iniciando deploy automatizado..."
Write-Host "[INFO] Imagem: $LatestImageName"
Write-Host ""

# Build da imagem
Write-Host "[BUILD] Fazendo build da imagem..."
docker build --no-cache -t $LatestImageName .

if ($LASTEXITCODE -ne 0) {
    Write-Error "[ERRO] Falha no build da imagem"
    exit 1
}

Write-Host "[OK] Build concluido"

# Push da imagem
Write-Host ""
Write-Host "[PUSH] Enviando imagem para o registry..."
docker push $LatestImageName

if ($LASTEXITCODE -ne 0) {
    Write-Error "[ERRO] Falha no push da imagem"
    exit 1
}

Write-Host "[OK] Imagem enviada"
Write-Host ""
Write-Host "========================================="
Write-Host "[CONCLUIDO] DEPLOY FINALIZADO!"
Write-Host "========================================="
Write-Host "[INFO] Latest: $LatestImageName"
Write-Host ""
Write-Host "[SERVIDOR] Para atualizar no servidor execute:"
Write-Host "   docker compose down"
Write-Host "   docker pull $LatestImageName"
Write-Host "   docker compose up -d"

