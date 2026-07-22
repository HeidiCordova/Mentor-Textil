# Jetson Orin - vision-event-detector
# Base: ubuntu:22.04 + NVIDIA L4T apt repo → libopencv-python 4.8.0 (CUDA + GStreamer)
# Requires --runtime=nvidia and nvidia container toolkit on the host.

FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1
ENV GST_PLUGIN_PATH=/usr/lib/aarch64-linux-gnu/gstreamer-1.0:/usr/lib/aarch64-linux-gnu/nvidia/gstreamer-1.0

# System deps + NVIDIA L4T repo for CUDA-enabled OpenCV 4.8.0
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl python3 python3-pip \
    && echo "deb [trusted=yes] https://repo.download.nvidia.com/jetson/common r36.4 main" \
       > /etc/apt/sources.list.d/nvidia-l4t.list \
    && apt-get update --allow-insecure-repositories \
    && apt-get install -y --no-install-recommends --allow-unauthenticated \
       libopencv \
       libopencv-python \
       libegl1 \
       python3.10-vpi3 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip3 install --no-cache-dir -r requirements.txt

COPY app/ ./app/

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD python3 -c "import urllib.request; urllib.request.urlopen('http://localhost:8001/health')" || exit 1

CMD ["python3", "-m", "app.main"]
