# Script para configurar acceso SSH sin password al Jetson
$jetsonUser = "orin"
$jetsonIP = "192.168.1.130"
$jetsonPassword = "123456"

Write-Host "=== Configurando acceso SSH al Jetson ===" -ForegroundColor Cyan
$publicKey = Get-Content "$env:USERPROFILE\.ssh\id_rsa.pub"
Write-Host "Clave publica:" -ForegroundColor Green
Write-Host $publicKey
Write-Host ""

$sshCommand = "mkdir -p ~/.ssh && chmod 700 ~/.ssh && echo '$publicKey' >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"

if (Get-Command plink -ErrorAction SilentlyContinue) {
    Write-Host "Usando plink para configuracion automatica..." -ForegroundColor Cyan
    echo y | plink -pw $jetsonPassword "$jetsonUser@$jetsonIP" $sshCommand
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Configuracion completada! Prueba: ssh $jetsonUser@$jetsonIP" -ForegroundColor Green
    }
} else {
    Write-Host "Conectate manualmente y ejecuta:" -ForegroundColor Yellow
    Write-Host "ssh $jetsonUser@$jetsonIP"
    Write-Host "Password: $jetsonPassword"
    Write-Host ""
    Write-Host $sshCommand -ForegroundColor White
}
