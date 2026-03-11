#!/bin/bash
# Automated cert-manager + Istio TLS setup for Pinz.
#
# This script installs cert-manager, creates an Istio-compatible IngressClass,
# provisions a Let's Encrypt ClusterIssuer, and requests a TLS certificate for
# the Istio ingress gateway secret.
#
# Usage:
#   DOMAIN=pinz.website EMAIL=admin@pinz.website ./setup-cert-manager.sh
#
# Optional env vars:
#   TLS_SECRET_NAME=pinz-tls        (default: pinz-tls)
#   ISTIO_NAMESPACE=istio-system    (default: istio-system)
#   CERT_MANAGER_NAMESPACE=cert-manager (default: cert-manager)
#   STAGING=true                    use Let's Encrypt staging

set -euo pipefail

DOMAIN="${DOMAIN:-}"
EMAIL="${EMAIL:-}"
TLS_SECRET_NAME="${TLS_SECRET_NAME:-pinz-tls}"
ISTIO_NAMESPACE="${ISTIO_NAMESPACE:-istio-system}"
CERT_MANAGER_NAMESPACE="${CERT_MANAGER_NAMESPACE:-cert-manager}"
STAGING="${STAGING:-false}"

ISSUER_NAME="letsencrypt-prod"
ACME_SERVER="https://acme-v02.api.letsencrypt.org/directory"
ACCOUNT_SECRET_NAME="letsencrypt-prod-account-key"
if [[ "$STAGING" == "true" ]]; then
  ISSUER_NAME="letsencrypt-staging"
  ACME_SERVER="https://acme-staging-v02.api.letsencrypt.org/directory"
  ACCOUNT_SECRET_NAME="letsencrypt-staging-account-key"
fi

if [[ -z "$DOMAIN" ]] || [[ -z "$EMAIL" ]]; then
  echo "Usage: DOMAIN=api.example.com EMAIL=admin@example.com ./setup-cert-manager.sh"
  echo "Optional: TLS_SECRET_NAME=pinz-tls  ISTIO_NAMESPACE=istio-system  CERT_MANAGER_NAMESPACE=cert-manager  STAGING=true"
  exit 1
fi

for cmd in kubectl helm; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "[ERROR] Required command not found: $cmd"; exit 1; }
done

kubectl get ns "$ISTIO_NAMESPACE" >/dev/null 2>&1 \
  || { echo "[ERROR] Kubernetes namespace not found: $ISTIO_NAMESPACE"; exit 1; }

install_cert_manager() {
  echo "[INFO] Installing/upgrading cert-manager..."
  helm repo add jetstack https://charts.jetstack.io >/dev/null 2>&1 || true
  helm repo update >/dev/null
  helm upgrade --install cert-manager jetstack/cert-manager \
    --namespace "$CERT_MANAGER_NAMESPACE" \
    --create-namespace \
    --set crds.enabled=true

  echo "[INFO] Waiting for cert-manager components..."
  kubectl rollout status deployment/cert-manager -n "$CERT_MANAGER_NAMESPACE" --timeout=300s
  kubectl rollout status deployment/cert-manager-cainjector -n "$CERT_MANAGER_NAMESPACE" --timeout=300s
  kubectl rollout status deployment/cert-manager-webhook -n "$CERT_MANAGER_NAMESPACE" --timeout=300s
}

apply_ingress_class() {
  echo "[INFO] Applying IngressClass for Istio..."
  kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: istio
spec:
  controller: istio.io/ingress-controller
EOF
}

apply_cluster_issuer() {
  echo "[INFO] Applying ClusterIssuer ${ISSUER_NAME}..."
  kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: ${ISSUER_NAME}
spec:
  acme:
    email: ${EMAIL}
    server: ${ACME_SERVER}
    privateKeySecretRef:
      name: ${ACCOUNT_SECRET_NAME}
    solvers:
      - http01:
          ingress:
            ingressClassName: istio
EOF
}

apply_certificate() {
  echo "[INFO] Requesting certificate for ${DOMAIN}..."
  kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: ${TLS_SECRET_NAME}
  namespace: ${ISTIO_NAMESPACE}
spec:
  secretName: ${TLS_SECRET_NAME}
  issuerRef:
    name: ${ISSUER_NAME}
    kind: ClusterIssuer
  commonName: ${DOMAIN}
  dnsNames:
    - ${DOMAIN}
EOF
}

cleanup_legacy_resources() {
  echo "[INFO] Removing legacy certbot/ACME resources..."
  kubectl delete deployment,service acme-challenge -n default --ignore-not-found >/dev/null
  kubectl delete deployment,service acme-challenge -n istio-system --ignore-not-found >/dev/null
  kubectl delete destinationrule acme-challenge -n default --ignore-not-found >/dev/null
  kubectl delete destinationrule acme-challenge -n istio-system --ignore-not-found >/dev/null
  kubectl delete peerauthentication acme-challenge -n default --ignore-not-found >/dev/null
  kubectl delete peerauthentication acme-challenge -n istio-system --ignore-not-found >/dev/null
  sudo rm -f \
    /etc/letsencrypt/renewal-hooks/pre/pinz-remove-redirect.sh \
    /etc/letsencrypt/renewal-hooks/post/pinz-restore-redirect.sh \
    /etc/letsencrypt/renewal-hooks/deploy/pinz-sync-istio-secret.sh 2>/dev/null || true
}

wait_for_certificate() {
  echo "[INFO] Waiting for certificate ${ISTIO_NAMESPACE}/${TLS_SECRET_NAME}..."
  kubectl wait --for=condition=Ready "certificate/${TLS_SECRET_NAME}" -n "$ISTIO_NAMESPACE" --timeout=600s
  kubectl get secret "$TLS_SECRET_NAME" -n "$ISTIO_NAMESPACE" >/dev/null
}

main() {
  install_cert_manager
  apply_ingress_class
  apply_cluster_issuer
  apply_certificate
  cleanup_legacy_resources
  wait_for_certificate

  echo ""
  echo "[OK] cert-manager setup complete."
  echo "    Domain : ${DOMAIN}"
  echo "    Issuer : ${ISSUER_NAME}"
  echo "    Secret : ${ISTIO_NAMESPACE}/${TLS_SECRET_NAME}"
}

main "$@"
