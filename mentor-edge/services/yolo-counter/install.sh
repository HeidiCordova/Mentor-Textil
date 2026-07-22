#!/usr/bin/env bash
# Instala yolo-counter como servicio nativo en el Jetson Orin (JetPack 6.x / L4T r36.x).
# Ejecutar como root o con sudo desde el directorio del repositorio.
# Uso: sudo bash mentor-edge/services/yolo-counter/install.sh
#
# Requisitos del host (JetPack 6.1 / L4T r36.4):
#   - CUDA 12.6, TensorRT 10.3, cuDNN 9.3, DeepStream 7.1 (pre-instalados via JetPack)
#   - Python 3.10
#
# Lo que instala este script:
#   1. libcusparseLt0-cuda-12  (necesaria para importar torch)
#   2. PyTorch 2.5.0 NVIDIA JP6.1 wheel (CUDA 12.6, aarch64)
#   3. torchvision compilado desde fuente (compatible con torch NVIDIA)
#   4. ultralytics 8.4+, supervision, onnx (para exportación TensorRT)
#   5. Servicio systemd yolo-counter

set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SERVICE_DIR="${REPO_DIR}/services/yolo-counter"
VENV_DIR="/opt/yolo-counter/venv"
SERVICE_USER="orin"
PYTORCH_WHEEL="https://developer.download.nvidia.com/compute/redist/jp/v61/pytorch/torch-2.5.0a0+872d972e41.nv24.08.17622132-cp310-cp310-linux_aarch64.whl"
CUSPARSELT_DEB="https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/arm64/libcusparselt0-cuda-12_0.8.1.1-1_arm64.deb"

echo "[yolo-counter] Repositorio: ${REPO_DIR}"
echo "[yolo-counter] Servicio en: ${SERVICE_DIR}"
echo "[yolo-counter] Virtualenv:  ${VENV_DIR}"

# 1. Dependencias del sistema
echo "[yolo-counter] Instalando dependencias del sistema..."
apt-get install -y --no-install-recommends \
    python3-venv python3-dev \
    libgl1 libglib2.0-0 \
    libjpeg-dev zlib1g-dev libpng-dev \
    git build-essential 2>/dev/null || true

# 2. libcusparseLt (requerida por torch 2.5.0 NVIDIA — no viene con JetPack por defecto)
if ! ldconfig -p 2>/dev/null | grep -q libcusparseLt; then
    echo "[yolo-counter] Instalando libcusparseLt0-cuda-12..."
    cd /tmp
    wget -q "${CUSPARSELT_DEB}" -O libcusparselt0-cuda-12.deb
    dpkg --force-depends -i libcusparselt0-cuda-12.deb
    echo /usr/lib/aarch64-linux-gnu/libcusparseLt/12 > /etc/ld.so.conf.d/cusparselt.conf
    ldconfig
    rm -f libcusparselt0-cuda-12.deb
fi

# 3. Crear virtualenv con acceso a paquetes del sistema (TensorRT python bindings, cv2)
mkdir -p /opt/yolo-counter
if [ ! -d "${VENV_DIR}" ]; then
    python3 -m venv "${VENV_DIR}" --system-site-packages
fi
source "${VENV_DIR}/bin/activate"
pip install --upgrade pip wheel

# 4. Instalar PyTorch CUDA para JetPack 6.1 (L4T r36.x, CUDA 12.6, Python 3.10)
if ! python -c "import torch; assert torch.cuda.is_available()" 2>/dev/null; then
    echo "[yolo-counter] Instalando PyTorch CUDA (wheel NVIDIA JP6.1)..."
    pip install "${PYTORCH_WHEEL}"
fi

# 5. Instalar torchvision desde fuente (compatible con torch 2.5.0a0 NVIDIA)
#    Los wheels de PyPI no son compatibles con el torch NVIDIA de JetPack
if ! python -c "import torchvision" 2>/dev/null; then
    echo "[yolo-counter] Compilando torchvision desde fuente (puede tardar 15-20 min)..."
    VISION_DIR="/tmp/torchvision-build"
    if [ ! -d "${VISION_DIR}" ]; then
        git clone --branch v0.20.0 --depth 1 https://github.com/pytorch/vision.git "${VISION_DIR}"
    fi
    cd "${VISION_DIR}"
    export FORCE_CUDA=1
    export PATH=/usr/local/cuda-12.6/bin:${PATH}
    export LD_LIBRARY_PATH=/usr/local/cuda-12.6/lib64:${LD_LIBRARY_PATH:-}
    python setup.py bdist_wheel
    pip install dist/torchvision-*.whl --no-deps
    cd /
fi

# 6. Instalar ultralytics, supervision y otras deps (sin actualizar torch)
echo "[yolo-counter] Instalando ultralytics y supervision..."
# Instalar sin deps primero para no conflictuar con el torch NVIDIA
pip install "ultralytics>=8.4.0" "supervision>=0.25.0" --no-deps
# Luego instalar las deps de ultralytics/supervision (excepto torch que ya está)
pip install \
    "numpy<2" \
    "matplotlib>=3.6.0" \
    "opencv-python>=4.6.0" \
    "pillow>=9.4" \
    "psutil>=5.8.0" \
    "polars>=0.20.0" \
    "ultralytics-thop>=2.0.18" \
    "scipy>=1.10.0" \
    "tqdm>=4.62.3" \
    "seaborn>=0.11.0" \
    "pyyaml>=5.3.1" \
    "requests>=2.26.0" \
    "defusedxml>=0.7.1" \
    "pyDeprecate<0.6.0,>=0.4.0"
# onnx para exportación TensorRT
pip install "onnx>=1.12.0,<2.0.0" "onnxslim>=0.1.71"

# 7. Verificar stack completo
echo "[yolo-counter] Verificando instalación..."
python << 'PYEOF'
import torch
assert torch.cuda.is_available(), "torch CUDA not available!"
print(f"  torch          {torch.__version__}  CUDA: {torch.cuda.is_available()}")
import torchvision; print(f"  torchvision    {torchvision.__version__}")
from ultralytics import YOLO; print("  ultralytics    OK")
import supervision as sv; print(f"  supervision    {sv.__version__}")
import cv2; print(f"  cv2            {cv2.__version__}")
import tensorrt; print(f"  tensorrt       {tensorrt.__version__}")
PYEOF

deactivate

# 8. Directorios de datos
mkdir -p /opt/yolo-counter/models /opt/yolo-counter/logs
chown -R "${SERVICE_USER}:${SERVICE_USER}" /opt/yolo-counter

# 9. Instalar servicio systemd
cp "${SERVICE_DIR}/yolo-counter.service" /etc/systemd/system/yolo-counter.service
systemctl daemon-reload
systemctl enable yolo-counter.service

echo ""
echo "[yolo-counter] Instalacion completa."
echo "  Iniciar:   systemctl start yolo-counter"
echo "  Logs:      journalctl -fu yolo-counter"
echo "  Estado:    systemctl status yolo-counter"
echo "  Health:    curl http://localhost:8006/health"
