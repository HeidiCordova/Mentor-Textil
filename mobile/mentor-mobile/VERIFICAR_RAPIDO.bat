@echo off
echo ============================================================
echo           VERIFICACION RAPIDA DE TRANSMISION
echo ============================================================
echo.

echo Obteniendo IP de esta computadora...
for /f "tokens=2 delims=:" %%a in ('ipconfig ^| findstr /c:"IPv4"') do (
    set IP=%%a
    goto :found
)
:found
set IP=%IP: =%
echo Tu IP es: %IP%

echo.
echo ============================================================
echo                    INSTRUCCIONES
echo ============================================================
echo.
echo 1. En tu app movil, configurar:
echo    IP: %IP%
echo    Puerto: 5000
echo.
echo 2. Presionar "Iniciar Transmision" en la app
echo.
echo 3. Este script mostrara si recibe datos
echo.
echo ============================================================
echo.

python verificar_transmision.py

pause