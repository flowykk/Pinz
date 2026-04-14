#!/usr/bin/env bash
# check-env.sh — проверяет, что все os.Getenv("VAR") из Go-кода прописаны
# в Helm deployment template соответствующего сервиса.
#
# Для каждого сервиса:
#   1. Извлекает имена переменных из os.Getenv("...") в Go-коде
#   2. Проверяет наличие каждой переменной в Helm deployment.yaml
#
# Исключения прописаны в .check-env-ignore (формат: SERVICE VAR_NAME).
# Exit code 1, если есть пропущенные переменные.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
IGNORE_FILE="$ROOT_DIR/scripts/.check-env-ignore"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

errors=0

is_ignored() {
    local svc="$1" var="$2"
    if [ -f "$IGNORE_FILE" ]; then
        grep -q "^${svc}[[:space:]]\+${var}$" "$IGNORE_FILE" 2>/dev/null && return 0
        grep -q "^\*[[:space:]]\+${var}$" "$IGNORE_FILE" 2>/dev/null && return 0
    fi
    return 1
}

check_service() {
    local svc_dir="$1"
    local helm_dir="$2"
    local svc_name="$3"

    local deployment="$ROOT_DIR/$helm_dir/templates/deployment.yaml"
    if [ ! -f "$deployment" ]; then
        echo -e "${YELLOW}SKIP${NC} $svc_name: no deployment.yaml at $deployment"
        return
    fi

    # Извлекаем уникальные имена переменных из os.Getenv("VAR") в Go-коде.
    local vars
    vars=$(grep -roh 'os\.Getenv("[^"]*")' "$ROOT_DIR/$svc_dir" --include='*.go' \
        | sed 's/os\.Getenv("//;s/")//' \
        | sort -u)

    if [ -z "$vars" ]; then
        echo -e "${GREEN}OK${NC}   $svc_name: no env vars found in code"
        return
    fi

    local missing=()
    for var in $vars; do
        if ! grep -q "$var" "$deployment"; then
            if ! is_ignored "$svc_name" "$var"; then
                missing+=("$var")
            fi
        fi
    done

    if [ ${#missing[@]} -eq 0 ]; then
        echo -e "${GREEN}OK${NC}   $svc_name: all env vars present in deployment.yaml"
    else
        echo -e "${RED}FAIL${NC} $svc_name: env vars missing from $helm_dir/templates/deployment.yaml:"
        for var in "${missing[@]}"; do
            echo "       - $var"
        done
        errors=$((errors + ${#missing[@]}))
    fi
}

echo "Checking environment variables consistency..."
echo ""

check_service "auth-service"        "helm/auth-service"  "auth-service"
check_service "trip-service"        "helm/trip-service"   "trip-service"
check_service "api-gateway-service" "helm/api-gateway"    "api-gateway"

echo ""
if [ "$errors" -gt 0 ]; then
    echo -e "${RED}Found $errors missing env var(s).${NC}"
    echo "When adding new env vars, update: Helm values, deployment.yaml, deploy.sh, helmfile, README."
    exit 1
else
    echo -e "${GREEN}All environment variables are consistent.${NC}"
fi
