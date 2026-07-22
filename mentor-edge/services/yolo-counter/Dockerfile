# Base: ubuntu:22.04 + CUDA runtime NVIDIA L4T repos + torch NVIDIA JP6 wheel
# Requiere: runtime=nvidia en docker-compose (para acceso a GPU devices).
FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
ENV PYTHONUNBUFFERED=1
ENV MODELS_DIR=/app/models
ENV LD_LIBRARY_PATH=/usr/local/cuda-12.6/targets/aarch64-linux/lib

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl gnupg python3 python3-pip libgl1 libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

# Paso 1: Agregar repos NVIDIA L4T (JetPack 6.x / r36.4) e instalar CUDA runtime
RUN curl -fsSL https://repo.download.nvidia.com/jetson/jetson-ota-public.asc \
      | gpg --dearmor -o /usr/share/keyrings/jetson-ota-public.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/jetson-ota-public.gpg] https://repo.download.nvidia.com/jetson/common r36.4 main" \
      > /etc/apt/sources.list.d/nvidia-l4t.list && \
    apt-get update && apt-get install -y --no-install-recommends \
      cuda-cudart-12-6 libcublas-12-6 cuda-nvtx-12-6 libcudnn9-cuda-12 \
      libcusparse-12-6 libcufft-12-6 libcurand-12-6 cuda-cupti-12-6 \
      libopenblas0 libgomp1 && \
    echo "/usr/local/cuda-12.6/targets/aarch64-linux/lib" > /etc/ld.so.conf.d/cuda.conf && \
    ldconfig && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Paso 2: PyTorch 2.5 CUDA para Jetson JP6.x (ARM64/aarch64, Python 3.10)
RUN pip3 install --no-cache-dir \
    "https://developer.download.nvidia.com/compute/redist/jp/v61/pytorch/torch-2.5.0a0+872d972e41.nv24.08.17622132-cp310-cp310-linux_aarch64.whl"

# Paso 3: Symlinks de CUDA libs en torch/lib/ para que _preload_cuda_deps las encuentre
RUN CUDA=/usr/local/cuda-12.6/targets/aarch64-linux/lib && \
    TL=/usr/local/lib/python3.10/dist-packages/torch/lib && \
    ln -sf $CUDA/libcudart.so.12    $TL/libcudart.so.12 && \
    ln -sf $CUDA/libcublas.so.12    $TL/libcublas.so.12 && \
    ln -sf $CUDA/libcublasLt.so.12  $TL/libcublasLt.so.12 && \
    ln -sf $CUDA/libnvToolsExt.so.1 $TL/libnvToolsExt.so.1 && \
    ln -sf $CUDA/libcusparse.so.12  $TL/libcusparse.so.12 && \
    ln -sf $CUDA/libcufft.so.11     $TL/libcufft.so.11 && \
    ln -sf $CUDA/libcurand.so.10    $TL/libcurand.so.10 && \
    ln -sf $CUDA/libcupti.so.12     $TL/libcupti.so.12

COPY requirements.txt .
# Paso 4: resto de dependencias con constraint para que pip NO reemplace torch CUDA
RUN echo "torch==2.5.0a0+872d972e41.nv24.8" > /constraints.txt && \
    pip3 install --no-cache-dir -r requirements.txt -c /constraints.txt

COPY app/ ./app/

EXPOSE 8006

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD python3 -c "import urllib.request; urllib.request.urlopen('http://localhost:8006/health')" || exit 1

CMD ["python3", "-m", "app.main"]
