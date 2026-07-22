@echo off
echo Abriendo puerto UDP 5000 en el Firewall de Windows...
echo.
netsh advfirewall firewall add rule name="GStreamer UDP 5000 IN" dir=in action=allow protocol=UDP localport=5000
netsh advfirewall firewall add rule name="GStreamer UDP 5000 OUT" dir=out action=allow protocol=UDP localport=5000
echo.
echo ✅ Reglas de firewall agregadas
echo.
pause