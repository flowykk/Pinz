#!/bin/bash
# Automated Certbot + Istio TLS setup for Pinz.
#
# Uses the standalone authenticator. The script temporarily removes the host
# port 80 redirect to the Istio ingress NodePort so certbot can bind :80,
# then restores the redirect and syncs the resulting certificate into the
# Istio TLS secret.
#
# Usage:
#   DOMAIN=pinz.website EMAIL=admin@pinz.website ./setup-certbot.sh
#
# Optional env vars:
#   TLS_SECRET_NAME=pinz-tls        (default: pinz-tls)
#   ISTIO_NAMESPACE=istio-system    (default: istio-system)
#   SKIP_ISSUE=true                 only (re)install hooks and sync existing cert
#   DRY_RUN=true                    pass --dry-run to certbot

set -euo pipefail

DOMAIN="${DOMAIN:-}"
EMAIL="${EMAIL:-}"
TLS_SECRET_NAME="${TLS_SECRET_NAME:-pinz-tls}"
ISTIO_NAMESPACE="${ISTIO_NAMESPACE:-istio-system}"
SKIP_ISSUE="${SKIP_ISSUE:-false}"
DRY_RUN="${DRY_RUN:-false}"

# ---------------------------------------------------------------------------
# Validation
# ---------------------------------------------------------------------------
if [[ -z "$DOMAIN" ]] || [[ -z "$EMAIL" ]]; then
  echo "Usage: DOMAIN=api.example.com EMAIL=admin@example.com ./setup-certbot.sh"
  echo "Optional: TLS_SECRET_NAME=pinz-tls  ISTIO_NAMESPACE=istio-system  SKIP_ISSUE=true  DRY_RUN=true"
  exit 1
fi

for cmd in kubectl sudo; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "[ERROR] Required command not found: $cmd"; exit 1; }
done

if ! command -v certbot >/dev/null 2>&1; then
  echo "[INFO] Installing certbot..."
  sudo apt-get update -q
  sudo apt-get install -y certbot
fi

kubectl get ns "$ISTIO_NAMESPACE" >/dev/null 2>&1 \
  || { echo "[ERROR] Kubernetes namespace not found: $ISTIO_NAMESPACE"; exit 1; }

# ---------------------------------------------------------------------------
# Istio ingress resources
# ---------------------------------------------------------------------------
apply_istio_ingress_resources() {
  echo "[INFO] Applying Istio Gateway and VirtualService for ${DOMAIN}..."
  kubectl apply -f - <<EOF
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: pinz-gateway
  namespace: default
spec:
  selector:
    istio: ingressgateway
  servers:
    - port:
        number: 80
        name: http
        protocol: HTTP
      hosts:
        - "${DOMAIN}"
    - port:
        number: 443
        name: https
        protocol: HTTPS
      tls:
        mode: SIMPLE
        credentialName: ${TLS_SECRET_NAME}
      hosts:
        - "${DOMAIN}"
---
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: api-gateway
  namespace: default
spec:
  hosts:
    - "${DOMAIN}"
  gateways:
    - pinz-gateway
  http:
    - match:
        - port: 80
          uri:
            prefix: /
      redirect:
        scheme: https
        redirectCode: 301
    - match:
        - port: 443
          uri:
            prefix: /health
      route:
        - destination:
            host: api-gateway.default.svc.cluster.local
            port:
              number: 8080
    - match:
        - port: 443
          uri:
            prefix: /
      route:
        - destination:
            host: api-gateway.default.svc.cluster.local
            port:
              number: 8080
EOF
}

cleanup_obsolete_acme_resources() {
  echo "[INFO] Removing obsolete ACME challenge resources..."
  kubectl delete deployment,service acme-challenge -n default --ignore-not-found >/dev/null
  kubectl delete deployment,service acme-challenge -n istio-system --ignore-not-found >/dev/null
  kubectl delete destinationrule acme-challenge -n default --ignore-not-found >/dev/null
  kubectl delete destinationrule acme-challenge -n istio-system --ignore-not-found >/dev/null
  kubectl delete peerauthentication acme-challenge -n default --ignore-not-found >/dev/null
  kubectl delete peerauthentication acme-challenge -n istio-system --ignore-not-found >/dev/null
}

get_http_nodeport() {
  kubectl get svc -n "$ISTIO_NAMESPACE" istio-ingressgateway \
    -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}'
}

remove_http_redirect() {
  local http_nodeport
  http_nodeport="$(get_http_nodeport)"

  if [[ -z "$http_nodeport" ]]; then
    echo "[WARN] Cannot resolve Istio HTTP NodePort; skipping redirect removal"
    return 0
  fi

  echo "[INFO] Temporarily removing port 80 redirect to NodePort ${http_nodeport}..."
  sudo iptables -t nat -D PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "$http_nodeport" 2>/dev/null || true
  sudo iptables -t nat -D OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "$http_nodeport" 2>/dev/null || true
}

restore_http_redirect() {
  local http_nodeport
  http_nodeport="$(get_http_nodeport)"

  if [[ -z "$http_nodeport" ]]; then
    echo "[WARN] Cannot resolve Istio HTTP NodePort; skipping redirect restore"
    return 0
  fi

  echo "[INFO] Restoring port 80 redirect to NodePort ${http_nodeport}..."
  if ! sudo iptables -t nat -C PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "$http_nodeport" &>/dev/null; then
    sudo iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "$http_nodeport"
  fi
  if ! sudo iptables -t nat -C OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "$http_nodeport" &>/dev/null; then
    sudo iptables -t nat -A OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "$http_nodeport"
  fi

  if command -v netfilter-persistent &>/dev/null; then
    sudo netfilter-persistent save >/dev/null
  fi
}

needs_standalone_migration() {
  local renewal_conf="/etc/letsencrypt/renewal/${DOMAIN}.conf"
  if sudo test -f "$renewal_conf" && ! sudo grep -Eq '^authenticator = standalone$' "$renewal_conf"; then
    return 0
  fi
  return 1
}

# ---------------------------------------------------------------------------
# Renewal hooks
# ---------------------------------------------------------------------------
write_renewal_hooks() {
  echo "[INFO] Installing certbot renewal hooks..."
  sudo mkdir -p /etc/letsencrypt/renewal-hooks/{pre,post,deploy}

  sudo tee /etc/letsencrypt/renewal-hooks/pre/pinz-remove-redirect.sh >/dev/null <<EOF
#!/bin/bash
set -euo pipefail
if [[ -f /etc/rancher/k3s/k3s.yaml ]]; then
  export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
fi
HTTP_NODEPORT=\$(kubectl get svc -n "${ISTIO_NAMESPACE}" istio-ingressgateway -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}' 2>/dev/null || true)
if [[ -n "\$HTTP_NODEPORT" ]]; then
  iptables -t nat -D PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "\$HTTP_NODEPORT" 2>/dev/null || true
  iptables -t nat -D OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "\$HTTP_NODEPORT" 2>/dev/null || true
fi
EOF

  sudo tee /etc/letsencrypt/renewal-hooks/post/pinz-restore-redirect.sh >/dev/null <<EOF
#!/bin/bash
set -euo pipefail
if [[ -f /etc/rancher/k3s/k3s.yaml ]]; then
  export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
fi
HTTP_NODEPORT=\$(kubectl get svc -n "${ISTIO_NAMESPACE}" istio-ingressgateway -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}' 2>/dev/null || true)
if [[ -n "\$HTTP_NODEPORT" ]]; then
  if ! iptables -t nat -C PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "\$HTTP_NODEPORT" &>/dev/null; then
    iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "\$HTTP_NODEPORT"
  fi
  if ! iptables -t nat -C OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "\$HTTP_NODEPORT" &>/dev/null; then
    iptables -t nat -A OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "\$HTTP_NODEPORT"
  fi
fi
if command -v netfilter-persistent &>/dev/null; then
  netfilter-persistent save >/dev/null
fi
EOF

  sudo tee /etc/letsencrypt/renewal-hooks/deploy/pinz-sync-istio-secret.sh >/dev/null <<EOF
#!/bin/bash
set -euo pipefail
if [[ -f /etc/rancher/k3s/k3s.yaml ]]; then
  export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
fi
CERT_DIR="/etc/letsencrypt/live/${DOMAIN}"
if [[ ! -f "\$CERT_DIR/fullchain.pem" ]] || [[ ! -f "\$CERT_DIR/privkey.pem" ]]; then
  echo "[deploy-hook] Cert files not found at \$CERT_DIR; skipping"
  exit 0
fi
kubectl create secret tls "${TLS_SECRET_NAME}" \
  --cert="\$CERT_DIR/fullchain.pem" \
  --key="\$CERT_DIR/privkey.pem" \
  -n "${ISTIO_NAMESPACE}" \
  --dry-run=client -o yaml | kubectl apply -f -
echo "[deploy-hook] TLS secret ${TLS_SECRET_NAME} synced in namespace ${ISTIO_NAMESPACE}"
EOF

  sudo chmod +x \
    /etc/letsencrypt/renewal-hooks/pre/pinz-remove-redirect.sh \
    /etc/letsencrypt/renewal-hooks/post/pinz-restore-redirect.sh \
    /etc/letsencrypt/renewal-hooks/deploy/pinz-sync-istio-secret.sh
}

# ---------------------------------------------------------------------------
# Certificate issuance
# ---------------------------------------------------------------------------
issue_certificate() {
  local certbot_args=(
    certonly --standalone --preferred-challenges http
    --non-interactive --agree-tos
    --email "$EMAIL"
    --cert-name "$DOMAIN"
    -d "$DOMAIN"
  )

  if needs_standalone_migration; then
    echo "[INFO] Existing renewal config is not standalone; forcing one-time reissue."
    certbot_args+=(--force-renewal)
  fi

  [[ "$DRY_RUN" == "true" ]] && certbot_args+=(--dry-run)

  echo "[INFO] Issuing certificate for $DOMAIN using standalone authenticator..."
  remove_http_redirect

  local certbot_status=0
  set +e
  sudo certbot "${certbot_args[@]}"
  certbot_status=$?
  set -e

  restore_http_redirect

  if [[ "$certbot_status" -ne 0 ]]; then
    return "$certbot_status"
  fi
}

# ---------------------------------------------------------------------------
# TLS secret sync
# ---------------------------------------------------------------------------
sync_tls_secret() {
  local cert_dir="/etc/letsencrypt/live/${DOMAIN}"
  local tmp_dir
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "'"${tmp_dir}"'"' RETURN

  if ! sudo test -f "${cert_dir}/fullchain.pem" || ! sudo test -f "${cert_dir}/privkey.pem"; then
    echo "[ERROR] Certificate not found at ${cert_dir}"
    exit 1
  fi

  sudo cp "${cert_dir}/fullchain.pem" "${tmp_dir}/fullchain.pem"
  sudo cp "${cert_dir}/privkey.pem" "${tmp_dir}/privkey.pem"
  sudo chown "$(id -u):$(id -g)" "${tmp_dir}/fullchain.pem" "${tmp_dir}/privkey.pem"
  chmod 600 "${tmp_dir}/fullchain.pem" "${tmp_dir}/privkey.pem"

  echo "[INFO] Syncing TLS secret ${TLS_SECRET_NAME} → namespace ${ISTIO_NAMESPACE}..."
  kubectl create secret tls "${TLS_SECRET_NAME}" \
    --cert="${tmp_dir}/fullchain.pem" \
    --key="${tmp_dir}/privkey.pem" \
    -n "${ISTIO_NAMESPACE}" \
    --dry-run=client -o yaml | kubectl apply -f -
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  cleanup_obsolete_acme_resources
  apply_istio_ingress_resources
  write_renewal_hooks

  if [[ "$SKIP_ISSUE" != "true" ]]; then
    issue_certificate
  fi

  sync_tls_secret

  echo "[INFO] Verifying certbot renew automation (dry-run)..."
  sudo certbot renew --dry-run

  echo ""
  echo "[OK] Certbot setup complete."
  echo "    Domain  : ${DOMAIN}"
  echo "    Secret  : ${ISTIO_NAMESPACE}/${TLS_SECRET_NAME}"
  echo "    Mode    : standalone"
}

main "$@"
