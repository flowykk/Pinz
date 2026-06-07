#!/usr/bin/env bash
# Первичная настройка сервера под Pinz AI Module.
# Ubuntu 22.04 / 24.04 LTS, root.
# Выполняет: системные пакеты, Docker, NVIDIA driver + container toolkit,
#            пользователь pinz, UFW, базовая структура каталогов.
#
# После окончания требуется reboot.
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
  echo "Run as root." >&2
  exit 1
fi

. /etc/os-release
if [[ "$ID" != "ubuntu" ]]; then
  echo "Only Ubuntu is supported by this script (detected: $ID)." >&2
  exit 1
fi

echo "[1/6] apt update + base packages"
apt-get update -y
DEBIAN_FRONTEND=noninteractive apt-get upgrade -y
DEBIAN_FRONTEND=noninteractive apt-get install -y \
  ca-certificates curl gnupg lsb-release ufw make git jq htop

echo "[2/6] docker"
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu $VERSION_CODENAME stable" \
  > /etc/apt/sources.list.d/docker.list
apt-get update -y
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
systemctl enable --now docker

echo "[3/6] NVIDIA driver + container toolkit"

# --- Driver ---
# Проверяем: провайдер мог предустановить драйвер (GPU Driver 535 и т.п.)
if nvidia-smi &>/dev/null; then
  echo "  NVIDIA driver already present: $(nvidia-smi --query-gpu=driver_version --format=csv,noheader | head -1)"
else
  echo "  Installing nvidia-driver-535 ..."
  # Устанавливаем напрямую через apt — ubuntu-drivers имеет известный баг
  # (AttributeError: 'NoneType'... depends_list_str) на Ubuntu 24.04
  DEBIAN_FRONTEND=noninteractive apt-get install -y \
    linux-headers-$(uname -r) \
    nvidia-driver-535 \
    nvidia-utils-535
fi

# --- NVIDIA Container Toolkit ---
# Используем актуальный URL (stable/deb/) вместо ${distribution}/
# — NVIDIA изменил структуру репозитория, ubuntu24.04 путь устарел
curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey \
  | gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
curl -fsSL https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list \
  | sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' \
  > /etc/apt/sources.list.d/nvidia-container-toolkit.list
apt-get update -y
apt-get install -y nvidia-container-toolkit
nvidia-ctk runtime configure --runtime=docker
systemctl restart docker

echo "[4/6] user 'pinz'"
if ! id -u pinz >/dev/null 2>&1; then
  adduser --disabled-password --gecos "Pinz AI" pinz
fi
usermod -aG docker pinz

# Authorized keys: если есть /root/.ssh/authorized_keys — копируем
if [[ -f /root/.ssh/authorized_keys ]]; then
  install -d -m 700 -o pinz -g pinz /home/pinz/.ssh
  install -m 600 -o pinz -g pinz /root/.ssh/authorized_keys /home/pinz/.ssh/authorized_keys
fi

echo "[5/6] /srv/pinz-ai + data dirs"
install -d -o pinz -g pinz /srv/pinz-ai
install -d -o pinz -g pinz /srv/pinz-ai/data/qdrant
install -d -o pinz -g pinz /srv/pinz-ai/data/meili

echo "[6/6] UFW: allow SSH only"
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow OpenSSH
ufw --force enable

cat <<EOF

============================================================
Bootstrap complete.

Next steps:
  1. reboot the server  (NVIDIA driver needs it)
  2. ssh pinz@<host>
  3. cd /srv/pinz-ai && git clone git@github.com:<org>/<repo>.git .
  4. cp AI/.env.example AI/.env.prod  (поставить NATS_URL и NATS_TOKEN)
  5. cd AI && make export-models
  6. docker compose --env-file .env.prod -f docker-compose.prod.yml up -d

Verify GPU passthrough after reboot:
  docker run --rm --gpus all nvidia/cuda:12.4.1-base-ubuntu22.04 nvidia-smi
============================================================
EOF
