#!/bin/bash
# Automated Certbot + Istio TLS setup for Pinz.
#
# Saves/restores the full iptables nat table around certbot --standalone
# so the Istio port-80 redirect is cleanly removed during the ACME challenge
# and always restored afterwards (even on error).
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

HTTP_NODEPORT=$(kubectl get svc -n "$ISTIO_NAMESPACE" istio-ingressgateway \
  -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}')
[[ -n "$HTTP_NODEPORT" ]] \
  || { echo "[ERROR] Cannot resolve Istio HTTP NodePort in namespace $ISTIO_NAMESPACE"; exit 1; }

echo "[INFO] Istio HTTP NodePort: $HTTP_NODEPORT"

# ---------------------------------------------------------------------------
# iptables helpers — save the entire nat table and restore from backup
# ---------------------------------------------------------------------------
_NAT_BACKUP=$(mktemp /tmp/pinz-nat-XXXXXX.rules)

save_nat_rules() {
  echo "[INFO] Saving iptables nat rules to $_NAT_BACKUP ..."
  sudo iptables-save -t nat > "$_NAT_BACKUP"
}

restore_nat_rules() {
  echo "[INFO] Restoring iptables nat rules from $_NAT_BACKUP ..."
  sudo iptables-restore < "$_NAT_BACKUP"
  rm -f "$_NAT_BACKUP"
}

# Remove every PREROUTING / OUTPUT rule that REDIRECTs port 80.
# Iterates by line number (safe: line numbers shift after each delete).
remove_port80_nat_redirects() {
  echo "[INFO] Removing port-80 NAT redirects so certbot can bind to port 80..."
  local line
  while line=$(sudo iptables -t nat -L PREROUTING --line-numbers -n \
      | awk '/REDIRECT/ && /dpt:80/ {print $1; exit}') && [[ -n "$line" ]]; do
    sudo iptables -t nat -D PREROUTING "$line"
  done
  while line=$(sudo iptables -t nat -L OUTPUT --line-numbers -n \
      | awk '/REDIRECT/ && /dpt:80/ {print $1; exit}') && [[ -n "$line" ]]; do
    sudo iptables -t nat -D OUTPUT "$line"
  done
}

# ---------------------------------------------------------------------------
# Renewal hooks (installed once; used on every automatic renewal)
# ---------------------------------------------------------------------------
write_renewal_hooks() {
  echo "[INFO] Installing certbot renewal hooks..."
  sudo mkdir -p /etc/letsencrypt/renewal-hooks/{pre,post,deploy}

  # pre-hook: free port 80 before certbot --standalone tries to bind
  sudo tee /etc/letsencrypt/renewal-hooks/pre/pinz-remove-redirect.sh >/dev/null <<'EOF'
#!/bin/bash
set -euo pipefail
line=""
while line=$(iptables -t nat -L PREROUTING --line-numbers -n \
    | awk '/REDIRECT/ && /dpt:80/ {print $1; exit}') && [[ -n "$line" ]]; do
  iptables -t nat -D PREROUTING "$line"
done
while line=$(iptables -t nat -L OUTPUT --line-numbers -n \
    | awk '/REDIRECT/ && /dpt:80/ {print $1; exit}') && [[ -n "$line" ]]; do
  iptables -t nat -D OUTPUT "$line"
done
echo "[pre-hook] Port-80 redirects removed"
EOF

  # post-hook: restore redirect to current Istio NodePort (re-resolves via kubectl each time)
  sudo tee /etc/letsencrypt/renewal-hooks/post/pinz-restore-redirect.sh >/dev/null <<EOF
#!/bin/bash
set -euo pipefail
HTTP_NODEPORT=\$(kubectl get svc -n ${ISTIO_NAMESPACE} istio-ingressgateway \
  -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}' 2>/dev/null || true)
if [[ -z "\${HTTP_NODEPORT:-}" ]]; then
  echo "[post-hook][ERROR] Cannot find Istio HTTP NodePort; redirect NOT restored"
  exit 1
fi
iptables -t nat -C PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "\$HTTP_NODEPORT" 2>/dev/null || \
  iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "\$HTTP_NODEPORT"
iptables -t nat -C OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "\$HTTP_NODEPORT" 2>/dev/null || \
  iptables -t nat -A OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "\$HTTP_NODEPORT"
command -v netfilter-persistent >/dev/null 2>&1 && netfilter-persistent save || true
echo "[post-hook] Restored port 80 → \$HTTP_NODEPORT"
EOF

  # deploy-hook: sync renewed certificate into the Istio TLS secret
  sudo tee /etc/letsencrypt/renewal-hooks/deploy/pinz-sync-istio-secret.sh >/dev/null <<EOF
#!/bin/bash
set -euo pipefail
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
    certonly --standalone --non-interactive --agree-tos
    --email "$EMAIL" -d "$DOMAIN"
    --preferred-challenges http
  )
  [[ "$DRY_RUN" == "true" ]] && certbot_args+=(--dry-run)

  echo "[INFO] Issuing certificate for $DOMAIN..."
  sudo certbot "${certbot_args[@]}"
}

# ---------------------------------------------------------------------------
# TLS secret sync
# ---------------------------------------------------------------------------
sync_tls_secret() {
  local cert_dir="/etc/letsencrypt/live/${DOMAIN}"
  [[ -f "${cert_dir}/fullchain.pem" ]] \
    || { echo "[ERROR] Certificate not found at ${cert_dir}"; exit 1; }
  echo "[INFO] Syncing TLS secret ${TLS_SECRET_NAME} → namespace ${ISTIO_NAMESPACE}..."
  kubectl create secret tls "${TLS_SECRET_NAME}" \
    --cert="${cert_dir}/fullchain.pem" \
    --key="${cert_dir}/privkey.pem" \
    -n "${ISTIO_NAMESPACE}" \
    --dry-run=client -o yaml | kubectl apply -f -
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  write_renewal_hooks

  if [[ "$SKIP_ISSUE" != "true" ]]; then
    save_nat_rules
    # Guarantee nat rules are restored even if something below fails
    trap 'restore_nat_rules' EXIT

    remove_port80_nat_redirects

    issue_certificate

    trap - EXIT          # disarm trap; restore explicitly so we see the log line
    restore_nat_rules
  fi

  sync_tls_secret

  echo "[INFO] Verifying certbot renew automation (dry-run)..."
  sudo certbot renew --dry-run

  echo ""
  echo "[OK] Certbot setup complete."
  echo "    Domain : ${DOMAIN}"
  echo "    Secret : ${ISTIO_NAMESPACE}/${TLS_SECRET_NAME}"
}

main "$@"
