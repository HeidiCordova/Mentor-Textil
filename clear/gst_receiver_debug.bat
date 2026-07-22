@echo off
echo Starting GStreamer receiver with debug info...
echo.
"C:\Program Files\GStreamer\1.0\mingw_x86_64\bin\gst-launch-1.0.exe" ^
  -v ^
  --gst-debug=udpsrc:5,rtph264depay:5 ^
  udpsrc address=0.0.0.0 port=5000 buffer-size=2097152 ^
  ! application/x-rtp,media=video,clock-rate=90000,encoding-name=H264,payload=96 ^
  ! rtph264depay ^
  ! h264parse ^
  ! avdec_h264 ^
  ! videoconvert ^
  ! autovideosink sync=false