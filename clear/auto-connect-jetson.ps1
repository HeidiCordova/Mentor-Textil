# Script para conectar automaticamente al Jetson
$host_key = "0x23,0x73,0x73,0x68,0x2d,0x65,0x64,0x32,0x35,0x35,0x31,0x39"

# Agregar host key al registro de plink
$regPath = "HKCU:\Software\SimonTatham\PuTTY\SshHostKeys"
if (!(Test-Path $regPath)) {
    New-Item -Path $regPath -Force | Out-Null
}

# Intentar registrar la clave
$keyName = "ed25519@22:192.168.1.130"
$keyValue = "0x3218b16488c47820b2627f6554396580782383c6b3727238bf27cb27b2524b74d"

try {
    New-ItemProperty -Path $regPath -Name $keyName -Value $keyValue -PropertyType String -Force -ErrorAction SilentlyContinue | Out-Null
} catch {}

Write-Host "Conectando al Jetson Orin..." -ForegroundColor Cyan
plink -pw "123456" orin@192.168.1.130
