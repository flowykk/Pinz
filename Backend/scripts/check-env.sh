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

## Phase 2: deployment.yaml → helmfile check.
## For each .Values.env.KEY / .Values.secret.KEY in deployment.yaml, verify
## that the helmfile release for that service sets the key.  Keys whose
## values.yaml default is non-empty and not a placeholder are skipped.

HELMFILE="$ROOT_DIR/helmfile.yaml.gotmpl"

# Extract the helmfile block for a given release name.
helmfile_release_block() {
    local release="$1"
    awk -v rel="$release" '
        $0 == "  - name: " rel { found=1; next }
        found && /^  - name: / { exit }
        found { print }
    ' "$HELMFILE"
}

check_helmfile() {
    local helm_dir="$1"
    local release_name="$2"
    local display_name="$3"

    if [ ! -f "$HELMFILE" ]; then
        echo -e "${YELLOW}SKIP${NC} helmfile check: $HELMFILE not found"
        return
    fi

    local deployment="$ROOT_DIR/$helm_dir/templates/deployment.yaml"
    local values="$ROOT_DIR/$helm_dir/values.yaml"
    if [ ! -f "$deployment" ] || [ ! -f "$values" ]; then
        return
    fi

    local block
    block=$(helmfile_release_block "$release_name")
    if [ -z "$block" ]; then
        echo -e "${YELLOW}SKIP${NC} $display_name: release '$release_name' not found in helmfile"
        return
    fi

    # Collect .Values.env.* and .Values.secret.* keys from deployment.yaml.
    local keys
    keys=$(grep -oE '\.Values\.(env|secret)\.[a-zA-Z0-9_]+' "$deployment" \
        | sed 's/\.Values\.//' | sort -u)

    local missing=()
    for entry in $keys; do
        local section="${entry%%.*}"   # env or secret
        local key="${entry#*.}"        # e.g. s3Bucket

        # Check default in values.yaml.  Skip if non-empty and not a placeholder.
        local default_val
        default_val=$(awk -v sec="$section" -v k="$key" '
            $0 == sec ":" { found=1; next }
            found && /^[^ ]/ { exit }
            found && $0 ~ "^  " k ":" {
                val=$0; sub(/^[^:]*: */, "", val)
                gsub(/^["'\''"]|["'\''"]$/, "", val)
                print val; exit
            }
        ' "$values")

        if [ -n "$default_val" ] && \
           [ "$default_val" != "REPLACE_IN_PRODUCTION" ] && \
           [ "$default_val" != "change-me-in-production" ]; then
            continue
        fi

        # Key needs a value from helmfile — check it's present in the release block.
        if ! echo "$block" | grep -q "${key}:"; then
            if ! is_ignored "$display_name" "$key"; then
                missing+=("${section}.${key}")
            fi
        fi
    done

    if [ ${#missing[@]} -eq 0 ]; then
        echo -e "${GREEN}OK${NC}   $display_name: all values set in helmfile"
    else
        echo -e "${RED}FAIL${NC} $display_name: values missing from helmfile.yaml.gotmpl (release '$release_name'):"
        for v in "${missing[@]}"; do
            echo "       - $v"
        done
        errors=$((errors + ${#missing[@]}))
    fi
}

echo "Checking environment variables consistency..."
echo ""
echo "Phase 1: Go code → deployment.yaml"

check_service "auth-service"        "helm/auth-service"  "auth-service"
check_service "trip-service"        "helm/trip-service"   "trip-service"
check_service "api-gateway-service" "helm/api-gateway"    "api-gateway"

echo ""
echo "Phase 2: deployment.yaml → helmfile"

check_helmfile "helm/auth-service"  "auth-service"  "auth-service"
check_helmfile "helm/trip-service"  "trip-service"   "trip-service"
check_helmfile "helm/api-gateway"   "api-gateway"    "api-gateway"

echo ""
if [ "$errors" -gt 0 ]; then
    echo -e "${RED}Found $errors missing env var(s).${NC}"
    echo "When adding new env vars, update: Helm values, deployment.yaml, deploy.sh, helmfile, README."
    exit 1
else
    echo -e "${GREEN}All environment variables are consistent.${NC}"
fi
