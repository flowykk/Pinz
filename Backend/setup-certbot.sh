#!/bin/bash
# Automated Certbot + Istio TLS setup for Pinz.
#
# Uses --webroot authenticator: certbot writes challenge tokens to a host
# directory that a dedicated nginx pod serves through Istio.  No iptables
# manipulation required.
#
# Usage:
#   DOMAIN=pinz.website EMAIL=admin@pinz.website ./setup-certbot.sh
#
# Optional env vars:
#   TLS_SECRET_NAME=pinz-tls        (default: pinz-tls)
#   ISTIO_NAMESPACE=istio-system    (default: istio-system)
#   ACME_WEBROOT=/var/www/acme-challenge  (default: /var/www/acme-challenge)
#   K8S_MANIFESTS_DIR=./k8s-istio   (default: ./k8s-istio)
#   SKIP_ISSUE=true                 only (re)install hooks and sync existing cert
#   DRY_RUN=true                    pass --dry-run to certbot

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOMAIN="${DOMAIN:-}"
EMAIL="${EMAIL:-}"
TLS_SECRET_NAME="${TLS_SECRET_NAME:-pinz-tls}"
ISTIO_NAMESPACE="${ISTIO_NAMESPACE:-istio-system}"
ACME_NAMESPACE="${ACME_NAMESPACE:-default}"
ACME_WEBROOT="${ACME_WEBROOT:-/var/www/acme-challenge}"
K8S_MANIFESTS_DIR="${K8S_MANIFESTS_DIR:-${SCRIPT_DIR}/k8s-istio}"
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

for cmd in kubectl; do
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
            prefix: /.well-known/acme-challenge/
      route:
        - destination:
            host: acme-challenge.${ACME_NAMESPACE}.svc.cluster.local
            port:
              number: 80
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

# ---------------------------------------------------------------------------
# Deploy acme-challenge nginx pod + required Istio policies
# ---------------------------------------------------------------------------
deploy_acme_handler() {
  echo "[INFO] Deploying acme-challenge handler..."
  kubectl apply -f "${K8S_MANIFESTS_DIR}/acme-challenge.yaml"
  kubectl apply -f "${K8S_MANIFESTS_DIR}/destination-rule.yaml"
  kubectl apply -f "${K8S_MANIFESTS_DIR}/peer-authentication.yaml"
  apply_istio_ingress_resources

  echo "[INFO] Waiting for acme-challenge pod to be ready..."
  kubectl rollout status deployment/acme-challenge -n "${ACME_NAMESPACE}" --timeout=60s

  echo "[INFO] Creating webroot directory: $ACME_WEBROOT"
  sudo mkdir -p "${ACME_WEBROOT}/.well-known/acme-challenge"
  sudo chmod -R 755 "$ACME_WEBROOT"
}

# ---------------------------------------------------------------------------
# Renewal hooks (installed once; used on every automatic renewal)
# ---------------------------------------------------------------------------
write_renewal_hooks() {
  echo "[INFO] Installing certbot renewal hooks..."
  sudo mkdir -p /etc/letsencrypt/renewal-hooks/{pre,post,deploy}
  # Remove legacy hooks from old standalone+iptables flow.
  sudo rm -f \
    /etc/letsencrypt/renewal-hooks/pre/pinz-remove-redirect.sh \
    /etc/letsencrypt/renewal-hooks/post/pinz-restore-redirect.sh

  # deploy-hook: sync renewed certificate into the Istio TLS secret
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

  sudo chmod +x /etc/letsencrypt/renewal-hooks/deploy/pinz-sync-istio-secret.sh
}

# ---------------------------------------------------------------------------
# Certificate issuance
# ---------------------------------------------------------------------------
issue_certificate() {
  local certbot_args=(
    certonly --webroot -w "$ACME_WEBROOT"
    --non-interactive --agree-tos
    --email "$EMAIL" -d "$DOMAIN"
  )
  [[ "$DRY_RUN" == "true" ]] && certbot_args+=(--dry-run)

  echo "[INFO] Issuing certificate for $DOMAIN (webroot: $ACME_WEBROOT)..."
  sudo certbot "${certbot_args[@]}"
}

# ---------------------------------------------------------------------------
# TLS secret sync
# ---------------------------------------------------------------------------
sync_tls_secret() {
  local cert_dir="/etc/letsencrypt/live/${DOMAIN}"
  local tmp_dir
  tmp_dir="$(mktemp -d)"
  # Expand path now so trap does not depend on local variable scope.
  trap 'rm -rf "'"${tmp_dir}"'"' RETURN

  if ! sudo test -f "${cert_dir}/fullchain.pem" || ! sudo test -f "${cert_dir}/privkey.pem"; then
    echo "[ERROR] Certificate not found at ${cert_dir}"
    exit 1
  fi

  # letsencrypt live/* is usually root-only; stage readable copies for kubectl.
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
  deploy_acme_handler
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
  echo "    Webroot : ${ACME_WEBROOT}"
}

main "$@"
