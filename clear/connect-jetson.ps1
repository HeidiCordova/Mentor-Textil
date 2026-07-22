# Script simple para conectarse al Jetson con plink
# La primera vez debes ejecutar manualmente: ssh orin@192.168.1.130 (ingresa password: 123456)
# Eso guardara la huella digital

$publicKey = Get-Content "$env:USERPROFILE\.ssh\id_rsa.pub" -Raw

Write-Host "Configurando SSH key en Jetson..." -ForegroundColor Cyan

# Usar plink para ejecutar comandos remotos
plink -batch -pw "123456" orin@192.168.1.130 "mkdir -p ~/.ssh && chmod 700 ~/.ssh && echo '$publicKey' >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && echo 'SSH key configurada!'"

if ($LASTEXITCODE -eq 0) {
    Write-Host ""  
    Write-Host "Conexion SSH configurada exitosamente!" -ForegroundColor Green
    Write-Host "Ahora puedes conectarte sin password con:" -ForegroundColor Cyan
    Write-Host "  ssh orin@192.168.1.130" -ForegroundColor White
} else {
    Write-Host ""
    Write-Host "Error: Primero debes conectarte manualmente para aceptar la huella digital" -ForegroundColor Yellow
    Write-Host "Ejecuta: ssh orin@192.168.1.130" -ForegroundColor White
    Write-Host "Password: 123456" -ForegroundColor Gray
    Write-Host "Luego ejecuta este script nuevamente" -ForegroundColor Yellow
}
