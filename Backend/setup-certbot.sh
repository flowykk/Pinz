#!/bin/bash

# Automated Certbot + Istio TLS setup for Pinz.
# - Issues Let's Encrypt cert for a domain
# - Keeps Istio TLS secret (pinz-tls) in sync on renewal
# - Temporarily disables iptables redirect on port 80 during ACME challenge

set -euo pipefail

DOMAIN="${DOMAIN:-}"
EMAIL="${EMAIL:-}"
TLS_SECRET_NAME="${TLS_SECRET_NAME:-pinz-tls}"
ISTIO_NAMESPACE="${ISTIO_NAMESPACE:-istio-system}"
SKIP_ISSUE="${SKIP_ISSUE:-false}"
DRY_RUN="${DRY_RUN:-false}"

if [[ -z "$DOMAIN" ]] || [[ -z "$EMAIL" ]]; then
  echo "Usage: DOMAIN=api.example.com EMAIL=admin@example.com ./setup-certbot.sh"
  echo "Optional env vars:"
  echo "  TLS_SECRET_NAME=pinz-tls"
  echo "  ISTIO_NAMESPACE=istio-system"
  echo "  SKIP_ISSUE=true          # only (re)install hooks and sync existing cert"
  echo "  DRY_RUN=true             # certbot --dry-run"
  exit 1
fi

if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl is required"
  exit 1
fi

if ! command -v sudo >/dev/null 2>&1; then
  echo "sudo is required"
  exit 1
fi

if ! command -v certbot >/dev/null 2>&1; then
  echo "[INFO] Installing certbot..."
  sudo apt update
  sudo apt install -y certbot
fi

if ! kubectl get ns "$ISTIO_NAMESPACE" >/dev/null 2>&1; then
  echo "Istio namespace not found: $ISTIO_NAMESPACE"
  exit 1
fi

get_http_nodeport() {
  kubectl get svc -n "$ISTIO_NAMESPACE" istio-ingressgateway \
    -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}'
}

write_hook_scripts() {
  echo "[INFO] Installing certbot renewal hooks..."

  sudo mkdir -p /etc/letsencrypt/renewal-hooks/pre
  sudo mkdir -p /etc/letsencrypt/renewal-hooks/post
  sudo mkdir -p /etc/letsencrypt/renewal-hooks/deploy

  # Pre-hook: remove redirect so certbot standalone can receive HTTP-01 traffic.
  sudo tee /etc/letsencrypt/renewal-hooks/pre/pinz-remove-redirect.sh >/dev/null <<'EOF'
#!/bin/bash
set -euo pipefail

remove_rule() {
  local table="$1"
  shift
  while sudo iptables -t "$table" -C "$@" &>/dev/null; do
    sudo iptables -t "$table" -D "$@"
  done
}

# Remove all known redirects to free port 80.
remove_rule nat PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 30569 || true
remove_rule nat OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port 30569 || true

# Also try dynamic removal for current Istio NodePort if available.
if command -v kubectl >/dev/null 2>&1; then
  HTTP_NODEPORT=$(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}' 2>/dev/null || true)
  if [[ -n "${HTTP_NODEPORT:-}" ]]; then
    remove_rule nat PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "$HTTP_NODEPORT" || true
    remove_rule nat OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "$HTTP_NODEPORT" || true
  fi
fi
EOF

  # Post-hook: restore redirect to current NodePort.
  sudo tee /etc/letsencrypt/renewal-hooks/post/pinz-restore-redirect.sh >/dev/null <<'EOF'
#!/bin/bash
set -euo pipefail

if ! command -v kubectl >/dev/null 2>&1; then
  exit 0
fi

HTTP_NODEPORT=$(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}')
if [[ -z "${HTTP_NODEPORT:-}" ]]; then
  exit 0
fi

if ! sudo iptables -t nat -C PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "$HTTP_NODEPORT" &>/dev/null; then
  sudo iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "$HTTP_NODEPORT"
fi
if ! sudo iptables -t nat -C OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "$HTTP_NODEPORT" &>/dev/null; then
  sudo iptables -t nat -A OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "$HTTP_NODEPORT"
fi

if command -v netfilter-persistent >/dev/null 2>&1; then
  sudo netfilter-persistent save
fi
EOF

  # Deploy-hook: sync renewed cert into k8s TLS secret.
  sudo tee /etc/letsencrypt/renewal-hooks/deploy/pinz-sync-istio-secret.sh >/dev/null <<EOF
#!/bin/bash
set -euo pipefail

DOMAIN="${DOMAIN}"
TLS_SECRET_NAME="${TLS_SECRET_NAME}"
ISTIO_NAMESPACE="${ISTIO_NAMESPACE}"

CERT_DIR="/etc/letsencrypt/live/\$DOMAIN"
if [[ ! -f "\$CERT_DIR/fullchain.pem" ]] || [[ ! -f "\$CERT_DIR/privkey.pem" ]]; then
  exit 0
fi

kubectl create secret tls "\$TLS_SECRET_NAME" \\
  --cert="\$CERT_DIR/fullchain.pem" \\
  --key="\$CERT_DIR/privkey.pem" \\
  -n "\$ISTIO_NAMESPACE" \\
  --dry-run=client -o yaml | kubectl apply -f -
EOF

  sudo chmod +x /etc/letsencrypt/renewal-hooks/pre/pinz-remove-redirect.sh
  sudo chmod +x /etc/letsencrypt/renewal-hooks/post/pinz-restore-redirect.sh
  sudo chmod +x /etc/letsencrypt/renewal-hooks/deploy/pinz-sync-istio-secret.sh
}

issue_certificate() {
  local certbot_args=(
    certonly
    --standalone
    --non-interactive
    --agree-tos
    --email "$EMAIL"
    -d "$DOMAIN"
    --preferred-challenges http
  )

  if [[ "$DRY_RUN" == "true" ]]; then
    certbot_args+=(--dry-run)
  fi

  echo "[INFO] Issuing certificate for $DOMAIN..."
  sudo certbot "${certbot_args[@]}"
}

sync_secret_now() {
  local cert_dir="/etc/letsencrypt/live/${DOMAIN}"
  if [[ ! -f "${cert_dir}/fullchain.pem" ]] || [[ ! -f "${cert_dir}/privkey.pem" ]]; then
    echo "Certificate files not found in ${cert_dir}"
    exit 1
  fi

  echo "[INFO] Syncing TLS secret ${TLS_SECRET_NAME} in namespace ${ISTIO_NAMESPACE}..."
  kubectl create secret tls "${TLS_SECRET_NAME}" \
    --cert="${cert_dir}/fullchain.pem" \
    --key="${cert_dir}/privkey.pem" \
    -n "${ISTIO_NAMESPACE}" \
    --dry-run=client -o yaml | kubectl apply -f -
}

main() {
  local http_nodeport
  http_nodeport="$(get_http_nodeport)"
  if [[ -z "$http_nodeport" ]]; then
    echo "Cannot resolve istio-ingressgateway HTTP NodePort in namespace ${ISTIO_NAMESPACE}"
    exit 1
  fi

  write_hook_scripts

  # Remove redirect before first issue (same as renewal pre-hook behavior).
  sudo /etc/letsencrypt/renewal-hooks/pre/pinz-remove-redirect.sh || true

  if [[ "$SKIP_ISSUE" != "true" ]]; then
    issue_certificate
  fi

  # Always restore redirect back.
  sudo /etc/letsencrypt/renewal-hooks/post/pinz-restore-redirect.sh || true

  sync_secret_now

  echo "[INFO] Testing certbot renew flow..."
  sudo certbot renew --dry-run

  echo "[OK] Certbot automation is configured."
  echo "    Domain: ${DOMAIN}"
  echo "    Secret: ${ISTIO_NAMESPACE}/${TLS_SECRET_NAME}"
}

main "$@"
