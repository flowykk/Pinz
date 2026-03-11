#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "[INFO] setup-certbot.sh is deprecated; forwarding to cert-manager setup."
exec "${SCRIPT_DIR}/setup-cert-manager.sh" "$@"
