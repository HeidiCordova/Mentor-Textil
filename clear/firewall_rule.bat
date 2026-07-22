@echo off
echo Adding Windows Firewall rule for UDP port 5000...
netsh advfirewall firewall add rule name="GStreamer UDP 5000" dir=in action=allow protocol=UDP localport=5000
echo.
echo Rule added. You can remove it later with:
echo netsh advfirewall firewall delete rule name="GStreamer UDP 5000"