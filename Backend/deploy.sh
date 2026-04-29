#!/bin/bash

# Pinz Backend Deployment Script
# Works both manually and from GitHub Actions CI/CD

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="${SCRIPT_DIR}"
ENV_FILE="${PROJECT_DIR}/.env"

# Default values
DOCKER_REGISTRY="${DOCKER_REGISTRY:-ghcr.io}"
DOCKER_REPO="${DOCKER_REPO:-dmitry-pr}"
IMAGE_TAG="${IMAGE_TAG:-latest}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running in CI
is_ci() {
    [[ -n "${CI:-}" ]] || [[ -n "${GITHUB_ACTIONS:-}" ]]
}

# Check if running on server
is_server() {
    [[ -f "/opt/pinz/Backend/deploy.sh" ]] || [[ "$PWD" == "/opt/pinz/Backend" ]]
}

# Load environment variables
load_env() {
    if [[ -f "$ENV_FILE" ]]; then
        log_info "Loading environment variables from $ENV_FILE"
        set -a
        source "$ENV_FILE"
        set +a
    else
        log_warning "Environment file $ENV_FILE not found"
    fi

    # Set defaults for required variables
    DOMAIN="${DOMAIN:-pinz.website}"
    SERVER_IP="${SERVER_IP:-host.docker.internal}"
    TLS_SECRET_NAME="${TLS_SECRET_NAME:-pinz-tls}"
    ISTIO_NAMESPACE="${ISTIO_NAMESPACE:-istio-system}"
    POSTGRES_PASSWORD="${POSTGRES_PASSWORD:?POSTGRES_PASSWORD must be set in .env or environment}"
    JWT_SECRET_KEY="${JWT_SECRET_KEY:?JWT_SECRET_KEY must be set in .env or environment}"
    # On k3s servers docker pull is not required; containerd pulls images directly.
    SKIP_PULL="${SKIP_PULL:-false}"
}

# Validate environment
validate_env() {
    log_info "Validating environment..."

    # Check required tools
    local required_tools=("kubectl" "helm" "helmfile")

    if is_ci || [[ "${SKIP_PULL:-false}" != "true" ]]; then
        required_tools+=("docker")
    fi

    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "$tool is not installed or not in PATH"
            exit 1
        fi
    done

    # Check Docker daemon only when docker-based auth/pull is required.
    if (is_ci || [[ "${SKIP_PULL:-false}" != "true" ]]) && ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi

    # Check Kubernetes connection
    log_info "Checking Kubernetes cluster connection..."
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        log_error "KUBECONFIG: ${KUBECONFIG:-not set}"
        log_error "kubectl version: $(kubectl version --client --short 2>/dev/null || echo 'kubectl not found')"
        log_error "Please ensure k3s is installed and running, and KUBECONFIG is set correctly"
        exit 1
    fi

    # Verify cluster is accessible
    if ! kubectl get nodes &> /dev/null; then
        log_error "Cannot access Kubernetes nodes"
        log_error "Cluster may not be ready or accessible"
        exit 1
    fi

    log_success "Environment validation passed"
}

# Check registry reachability (quick TCP probe, no auth required).
registry_reachable() {
    timeout 5 bash -c ">/dev/tcp/ghcr.io/443" 2>/dev/null
}

# Setup Docker registry authentication.
# On a k3s server this is best-effort: k3s uses containerd and pulls images
# directly when pods start, so docker login / docker pull are not required.
# In CI (GitHub Actions) login is mandatory because the build step pushes.
setup_docker_auth() {
    if is_ci; then
        log_info "Setting up Docker authentication for CI"
        echo "${GITHUB_TOKEN:-}" | docker login "$DOCKER_REGISTRY" -u "${GITHUB_ACTOR:-}" --password-stdin
        return
    fi

    if [[ "${SKIP_PULL:-false}" == "true" ]]; then
        log_info "SKIP_PULL=true — skipping Docker auth"
        return
    fi

    if ! registry_reachable; then
        log_warning "Cannot reach $DOCKER_REGISTRY (network timeout)."
        log_warning "k3s will pull images directly via containerd when pods start."
        log_warning "Set SKIP_PULL=true to suppress this check in future runs."
        return
    fi

    log_info "Setting up Docker authentication for server"
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        echo "$GITHUB_TOKEN" | docker login "$DOCKER_REGISTRY" -u "${GITHUB_ACTOR:-$USER}" --password-stdin
    else
        log_warning "GITHUB_TOKEN not set — skipping docker login (registry is public)"
    fi
}

# Pre-pull Docker images into the Docker daemon cache.
# On k3s this is optional: containerd handles pulls independently.
# Skipped automatically when registry is unreachable or SKIP_PULL=true.
pull_images() {
    if [[ "${SKIP_PULL:-false}" == "true" ]]; then
        log_info "SKIP_PULL=true — skipping image pull"
        return
    fi

    if ! registry_reachable; then
        log_warning "Registry unreachable — skipping docker pull (k3s will pull on pod start)"
        return
    fi

    local api_gateway_image="${DOCKER_REGISTRY}/${DOCKER_REPO}/pinz-api-gateway:${IMAGE_TAG}"
    local auth_service_image="${DOCKER_REGISTRY}/${DOCKER_REPO}/pinz-auth-service:${IMAGE_TAG}"
    local trip_service_image="${DOCKER_REGISTRY}/${DOCKER_REPO}/pinz-trip-service:${IMAGE_TAG}"
    local statistics_service_image="${DOCKER_REGISTRY}/${DOCKER_REPO}/pinz-statistics-service:${IMAGE_TAG}"

    log_info "Pulling Docker images..."
    docker pull "$api_gateway_image" || log_warning "Pull failed for $api_gateway_image (non-fatal on k3s)"
    docker pull "$auth_service_image" || log_warning "Pull failed for $auth_service_image (non-fatal on k3s)"
    docker pull "$trip_service_image" || log_warning "Pull failed for $trip_service_image (non-fatal on k3s)"
    docker pull "$statistics_service_image" || log_warning "Pull failed for $statistics_service_image (non-fatal on k3s)"

    log_success "Images pull step complete"
}

# Select Helmfile configuration
select_helmfile() {
    HELMFILE_CONFIG="${PROJECT_DIR}/helmfile.yaml.gotmpl"
    ISTIO_CONFIG_DIR="${PROJECT_DIR}/k8s-istio"

    if [[ ! -f "$HELMFILE_CONFIG" ]]; then
        log_error "Helmfile config not found: $HELMFILE_CONFIG"
        exit 1
    fi

    log_info "Using Helmfile config: $HELMFILE_CONFIG"
}

# Deploy application
deploy_app() {
    log_info "Starting deployment..."

    # Export env vars so helmfile.yaml.gotmpl can read them via env().
    # Values come from load_env (.env) or the caller's environment.
    export DOCKER_REGISTRY="$DOCKER_REGISTRY"
    export DOCKER_REPO="$DOCKER_REPO"
    export IMAGE_TAG="$IMAGE_TAG"
    # Per-service image tags (CD sets these individually; fall back to IMAGE_TAG)
    export API_GATEWAY_TAG="${API_GATEWAY_TAG:-${IMAGE_TAG}}"
    export AUTH_SERVICE_TAG="${AUTH_SERVICE_TAG:-${IMAGE_TAG}}"
    export TRIP_SERVICE_TAG="${TRIP_SERVICE_TAG:-${IMAGE_TAG}}"
    export STATISTICS_SERVICE_TAG="${STATISTICS_SERVICE_TAG:-${IMAGE_TAG}}"
    export NOTIFICATION_SERVICE_TAG="${NOTIFICATION_SERVICE_TAG:-${IMAGE_TAG}}"
    export SERVER_IP="${SERVER_IP:-host.docker.internal}"
    export POSTGRES_PASSWORD="${POSTGRES_PASSWORD}"
    export JWT_SECRET_KEY="${JWT_SECRET_KEY}"
    # notification-service: APNS + SMTP env (optional, service works without them)
    export APNS_KEY_ID="${APNS_KEY_ID:-}"
    export APNS_TEAM_ID="${APNS_TEAM_ID:-}"
    export APNS_BUNDLE_ID="${APNS_BUNDLE_ID:-}"
    export APNS_KEY_BASE64="${APNS_KEY_BASE64:-}"
    export APNS_PRODUCTION="${APNS_PRODUCTION:-false}"
    export SMTP_HOST="${SMTP_HOST:-}"
    export SMTP_PORT="${SMTP_PORT:-587}"
    export SMTP_FROM="${SMTP_FROM:-}"
    export SMTP_USERNAME="${SMTP_USERNAME:-}"
    export SMTP_PASSWORD="${SMTP_PASSWORD:-}"
    # S3 object storage for auth-service (avatars) and trip-service (media)
    export S3_ENDPOINT="${S3_ENDPOINT:-}"
    export S3_REGION="${S3_REGION:-}"
    export S3_BUCKET="${S3_BUCKET:-}"
    export S3_ACCESS_KEY="${S3_ACCESS_KEY:-}"
    export S3_SECRET_KEY="${S3_SECRET_KEY:-}"
    export S3_PRESIGN_TTL="${S3_PRESIGN_TTL:-}"
    # statistics-service geocoding (optional; empty = use default BigDataCloud free endpoint)
    export GEOCODING_BASE_URL="${GEOCODING_BASE_URL:-}"
    export GEOCODING_API_KEY="${GEOCODING_API_KEY:-}"
    # api-gateway: base URL used to build share_url in trip responses (ТЗ 3.4).
    # Empty falls back to gateway's internal default (https://pinz.website/trips).
    export TRIP_SHARE_LINK_BASE="${TRIP_SHARE_LINK_BASE:-}"

    cd "$PROJECT_DIR"

    # DEPLOY_SERVICES controls which helm releases are applied:
    #   unset/empty — deploy all releases (manual run, backward compat)
    #   "none"      — skip helmfile entirely (only infra/istio/observability)
    #   "a,b"       — deploy only listed releases
    if [[ "${DEPLOY_SERVICES:-}" == "none" ]]; then
        log_info "DEPLOY_SERVICES=none — skipping helmfile apply (infra-only deploy)"
    elif [[ -n "${DEPLOY_SERVICES:-}" ]]; then
        log_info "Selective deploy: ${DEPLOY_SERVICES}"
        local -a selector_args=()
        for svc in ${DEPLOY_SERVICES//,/ }; do
            selector_args+=(-l "name=${svc}")
        done
        helmfile -f "$HELMFILE_CONFIG" "${selector_args[@]}" apply
    else
        helmfile -f "$HELMFILE_CONFIG" apply
    fi

    log_success "Application deployed successfully"
}

# Start or ensure infrastructure (Postgres x2, Redis) via Docker Compose.
# Idempotent: already running containers are left as-is.
start_infra() {
    local compose_file="${PROJECT_DIR}/docker-compose.infra.yml"
    if [[ ! -f "$compose_file" ]]; then
        log_warning "Infra compose not found: $compose_file — skipping (assume external DB/Redis)"
        return 0
    fi
    if ! command -v docker &>/dev/null; then
        log_warning "docker not found — skipping infra start (assume external DB/Redis)"
        return 0
    fi
    if ! docker info &>/dev/null; then
        log_warning "Docker daemon not reachable — skipping infra start"
        return 0
    fi
    log_info "Starting infrastructure (Postgres x2, Redis) if not already running..."
    (cd "$PROJECT_DIR" && docker compose -f docker-compose.infra.yml up -d) || {
        log_warning "Infra start failed (non-fatal); ensure DB and Redis are available"
        return 0
    }
    log_success "Infrastructure ready"
}

# Apply external services for infrastructure access
apply_external_services() {
    local external_services_file="${PROJECT_DIR}/k8s/k8s-external-services.yaml"

    if [[ ! -f "$external_services_file" ]]; then
        log_warning "External services file not found: $external_services_file"
        log_warning "Skipping external services apply"
        return 0
    fi

    if [[ -z "${SERVER_IP:-}" ]]; then
        log_error "SERVER_IP is not set — cannot apply external services (DB_HOST/REDIS_ADDR would be unresolvable)"
        exit 1
    fi

    log_info "Applying external services from: $external_services_file (SERVER_IP=${SERVER_IP})"
    SERVER_IP="$SERVER_IP" envsubst '${SERVER_IP}' < "$external_services_file" | kubectl apply -f -
    log_success "External services applied"
}

# Apply Kubernetes NetworkPolicy manifests (ingress segmentation layer).
apply_network_policies() {
    local network_policy_file="${PROJECT_DIR}/k8s/k8s-network-policies.yaml"

    if [[ ! -f "$network_policy_file" ]]; then
        log_warning "NetworkPolicy file not found: $network_policy_file"
        log_warning "Skipping NetworkPolicy apply"
        return 0
    fi

    log_info "Applying NetworkPolicy resources from: $network_policy_file"
    kubectl apply -f "$network_policy_file"
    log_success "NetworkPolicy resources applied"
}

# Apply observability stack (OTel Collector, Tempo, Prometheus, Loki, Grafana).
# On VPS, GF_SERVER_ROOT_URL is set to https://grafana.pinz.website for correct links/redirects.
apply_observability() {
    local observability_file="${PROJECT_DIR}/k8s/k8s-observability.yaml"

    if [[ ! -f "$observability_file" ]]; then
        log_warning "Observability manifest not found: $observability_file"
        log_warning "Skipping observability stack apply"
        return 0
    fi

    log_info "Applying observability stack from: $observability_file"
    if is_server; then
        export GRAFANA_ROOT_URL="${GRAFANA_ROOT_URL:-https://grafana.pinz.website}"
        sed "s|http://localhost:3000|${GRAFANA_ROOT_URL}|g" "$observability_file" | kubectl apply -f -
    else
        kubectl apply -f "$observability_file"
    fi
    log_success "Observability stack applied"
}

# Apply Istio resources from static manifests.
apply_istio_routing() {
    if [[ ! -d "$ISTIO_CONFIG_DIR" ]]; then
        log_warning "Istio config directory not found: $ISTIO_CONFIG_DIR"
        log_warning "Skipping Istio resource apply"
        return 0
    fi

    if ! kubectl get ns "$ISTIO_NAMESPACE" &>/dev/null; then
        log_warning "Istio namespace (${ISTIO_NAMESPACE}) not found"
        log_warning "Skipping Istio resource apply"
        return 0
    fi

    log_info "Applying Istio resources from: $ISTIO_CONFIG_DIR"
    kubectl apply -f "$ISTIO_CONFIG_DIR"
    log_success "Istio resources applied"

    if ! kubectl get secret "$TLS_SECRET_NAME" -n "$ISTIO_NAMESPACE" &>/dev/null; then
        log_warning "TLS secret ${ISTIO_NAMESPACE}/${TLS_SECRET_NAME} not found. HTTPS on port 443 will not work until secret is created."
        log_warning "Run setup-cert-manager.sh (or legacy setup-certbot.sh) or create secret manually."
    fi
}

# Ensure standard ports are forwarded to Istio ingress NodePorts.
setup_port_forwarding() {
    if ! is_server; then
        return 0
    fi

    if ! command -v sudo &>/dev/null; then
        log_warning "sudo not found, skipping iptables setup"
        return 0
    fi

    local http_nodeport
    local https_nodeport
    http_nodeport=$(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}')
    https_nodeport=$(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath='{.spec.ports[?(@.port==443)].nodePort}')

    if [[ -z "$http_nodeport" ]] || [[ -z "$https_nodeport" ]]; then
        log_warning "Cannot resolve Istio ingress NodePorts, skipping iptables setup"
        return 0
    fi

    log_info "Ensuring port forwarding 80→${http_nodeport}, 443→${https_nodeport}"

    # Remove legacy broad redirects first. They also catch pod egress traffic
    # that happens to target external port 80/443, which breaks outbound HTTPS.
    sudo iptables -t nat -D PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "$http_nodeport" 2>/dev/null || true
    sudo iptables -t nat -D PREROUTING -p tcp --dport 443 -j REDIRECT --to-port "$https_nodeport" 2>/dev/null || true

    # Only redirect traffic addressed to this node. Do not intercept forwarded
    # pod traffic to arbitrary internet destinations.
    if ! sudo iptables -t nat -C PREROUTING -m addrtype --dst-type LOCAL -p tcp --dport 80 -j REDIRECT --to-port "$http_nodeport" &>/dev/null; then
        sudo iptables -t nat -A PREROUTING -m addrtype --dst-type LOCAL -p tcp --dport 80 -j REDIRECT --to-port "$http_nodeport"
    fi
    if ! sudo iptables -t nat -C PREROUTING -m addrtype --dst-type LOCAL -p tcp --dport 443 -j REDIRECT --to-port "$https_nodeport" &>/dev/null; then
        sudo iptables -t nat -A PREROUTING -m addrtype --dst-type LOCAL -p tcp --dport 443 -j REDIRECT --to-port "$https_nodeport"
    fi
    if ! sudo iptables -t nat -C OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "$http_nodeport" &>/dev/null; then
        sudo iptables -t nat -A OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "$http_nodeport"
    fi
    if ! sudo iptables -t nat -C OUTPUT -p tcp -d 127.0.0.1 --dport 443 -j REDIRECT --to-port "$https_nodeport" &>/dev/null; then
        sudo iptables -t nat -A OUTPUT -p tcp -d 127.0.0.1 --dport 443 -j REDIRECT --to-port "$https_nodeport"
    fi

    if command -v netfilter-persistent &>/dev/null; then
        sudo netfilter-persistent save
    fi
}

# Wait until a Deployment object exists in the cluster (it may take a few seconds
# after helmfile apply for the API server to register the resource).
wait_for_deployment_object() {
    local deploy="$1"
    local max_wait=60
    local elapsed=0

    log_info "Waiting for Deployment object '$deploy' to appear..."
    until kubectl get deployment "$deploy" &>/dev/null; do
        if [[ $elapsed -ge $max_wait ]]; then
            log_error "Deployment object '$deploy' did not appear within ${max_wait}s."
            log_error "helmfile apply may have failed. Check helmfile output above."
            return 1
        fi
        sleep 3
        (( elapsed += 3 ))
    done
    log_info "Deployment object '$deploy' found after ${elapsed}s."
}

# Wait for deployment to be ready
wait_for_deployment() {
    log_info "Waiting for deployment to be ready..."

    local deployments
    if [[ "${DEPLOY_SERVICES:-}" == "none" ]]; then
        log_info "No service releases deployed — skipping rollout wait"
        return 0
    elif [[ -n "${DEPLOY_SERVICES:-}" ]]; then
        IFS=',' read -ra deployments <<< "$DEPLOY_SERVICES"
    else
        deployments=("api-gateway" "auth-service" "trip-service" "statistics-service")
    fi

    for deploy in "${deployments[@]}"; do
        # Ensure the Deployment object itself exists before calling rollout status.
        wait_for_deployment_object "$deploy" || return 1

        log_info "Checking rollout status for: $deploy"

        if ! kubectl rollout status "deployment/${deploy}" --timeout=600s; then
            log_error "Rollout timed out for: $deploy"
            log_error "--- Pod list ---"
            kubectl get pods -l "app=${deploy}" -o wide
            log_error "--- Events ---"
            kubectl describe deployment "${deploy}" | tail -30
            local pending_pod
            pending_pod=$(kubectl get pods -l "app=${deploy}" \
                --field-selector='status.phase!=Running' \
                -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
            if [[ -n "$pending_pod" ]]; then
                log_error "--- Pending pod logs: $pending_pod ---"
                kubectl describe pod "$pending_pod"
            fi
            return 1
        fi

        log_success "Rollout complete: $deploy"
    done

    log_success "All deployments are ready"
}

# Health check
health_check() {
    log_info "Performing health checks..."

    local api_url=""
    if is_server; then
        # Prefer domain name; fall back to SERVER_IP then node internal IP.
        local host="${DOMAIN:-${SERVER_IP:-$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')}}"
        api_url="http://$host"
        log_info "Using server access: $api_url (port 80 → Istio NodePort via iptables)"
    else
        api_url="http://localhost:8080"
    fi

    # Always send Host: pinz.website so Istio VirtualService matches regardless
    # of whether the URL is a domain name, IP, or NodePort.
    local host_header="${DOMAIN:-pinz.website}"

    local max_attempts=30
    local attempt=1

    while [[ $attempt -le $max_attempts ]]; do
        if curl -f -s -H "Host: ${host_header}" "$api_url/health" &> /dev/null; then
            log_success "Health check passed: $api_url/health (Host: ${host_header})"
            return 0
        fi

        log_info "Health check attempt $attempt/$max_attempts failed, retrying..."
        sleep 10
        ((attempt++))
    done

    log_error "Health check failed after $max_attempts attempts"
    log_error "Tip: check that Gateway/VirtualService are applied: kubectl get gateways.networking.istio.io,virtualservice -n default"
    return 1
}

# Show deployment status
show_status() {
    log_info "Deployment status:"

    echo ""
    echo "=== Pods ==="
    kubectl get pods

    echo ""
    echo "=== Services ==="
    kubectl get services

    echo ""
    echo "=== Deployments ==="
    kubectl get deployments

    echo ""
    echo "=== Resource Usage ==="
    kubectl top pods 2>/dev/null || echo "kubectl top not available"
}

# Cleanup function
cleanup() {
    local exit_code=$?
    if [[ $exit_code -ne 0 ]]; then
        log_error "Deployment failed with exit code $exit_code"
        show_status
    fi
}

# Main function
main() {
    trap cleanup EXIT

    log_info "🚀 Starting Pinz Backend deployment"
    log_info "Image Tag: $IMAGE_TAG"
    log_info "Running in CI: $(is_ci && echo 'Yes' || echo 'No')"
    log_info "Running on server: $(is_server && echo 'Yes' || echo 'No')"

    # Load environment
    load_env

    # Validate environment
    validate_env

    # Setup Docker auth
    setup_docker_auth

    # Pull images
    pull_images

    # Select deployment configs
    select_helmfile

    # Start infra (Postgres x2, Redis) if not already running
    start_infra

    # Apply external services for infrastructure access
    apply_external_services

    # Deploy application
    deploy_app

    # Apply observability stack (Grafana, Tempo, Loki, etc.)
    apply_observability

    # Apply Istio ingress routing resources (if present)
    apply_istio_routing

    # Apply Kubernetes NetworkPolicy ingress segmentation
    apply_network_policies

    # Sync port forwarding rules for standard external ports
    setup_port_forwarding

    # Wait for readiness
    wait_for_deployment

    # Health check
    health_check

    # Show status
    show_status

    log_success "🎉 Deployment completed successfully!"

    # Show access information
    if is_server; then
        local host="${DOMAIN:-${SERVER_IP:-$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')}}"
        log_info ""
        log_info "External Access Information:"
        log_info "  HTTP:   http://$host/health"
        log_info "  HTTPS:  https://$host/health"
        log_info "  Swagger: https://$host/swagger/index.html"
        log_info "  Grafana: https://grafana.pinz.website"
        log_info "  (port 80/443 → Istio NodePort via iptables)"
    else
        log_info "Application is available at: http://localhost:8080"
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --image-tag|-t)
            IMAGE_TAG="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Pinz Backend deployment script"
            echo ""
            echo "Options:"
            echo "  -t, --image-tag TAG      Docker image tag [default: latest]"
            echo "  -h, --help               Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  IMAGE_TAG                Same as --image-tag"
            echo "  SERVER_IP                Server IP address for database access"
            echo "  POSTGRES_PASSWORD        Database password"
            echo "  JWT_SECRET_KEY           JWT secret key"
            echo "  SMTP_HOST                Notification service: SMTP host (optional)"
            echo "  SMTP_PORT                Notification service: SMTP port (default: 587)"
            echo "  SMTP_USERNAME            Notification service: SMTP username (optional)"
            echo "  SMTP_PASSWORD            Notification service: SMTP password (optional)"
            echo "  SMTP_FROM                Notification service: sender email address (optional)"
            echo "  APNS_KEY_ID              Notification service: APNS .p8 key ID (optional)"
            echo "  APNS_TEAM_ID             Notification service: Apple team ID (optional)"
            echo "  APNS_BUNDLE_ID           Notification service: iOS app bundle id (optional)"
            echo "  APNS_KEY_BASE64          Notification service: .p8 key contents (base64)"
            echo "  APNS_PRODUCTION          Notification service: 'true' for prod Apple host (default: false)"
            echo "  S3_ENDPOINT              S3 API endpoint (auth + trip, optional)"
            echo "  S3_REGION                S3 region (auth + trip, optional)"
            echo "  S3_BUCKET                S3 bucket (auth + trip; empty = disabled)"
            echo "  S3_ACCESS_KEY            S3 access key (auth + trip, optional)"
            echo "  S3_SECRET_KEY            S3 secret key (auth + trip, optional)"
            echo "  S3_PRESIGN_TTL           S3 presign TTL, e.g. 15m (auth + trip, optional)"
            echo "  GEOCODING_BASE_URL       Statistics service: BigDataCloud API base URL (optional)"
            echo "  GEOCODING_API_KEY        Statistics service: BigDataCloud API key (optional)"
            echo "  TRIP_SHARE_LINK_BASE     API gateway: base URL for share_url in trip responses (default: https://pinz.website/trips)"
            echo "  SKIP_PULL=true           Skip docker auth/pull (k3s pulls via containerd)"
            echo ""
            echo "Examples:"
            echo "  $0                                    # Deploy with latest tag"
            echo "  $0 -t v1.2.3                         # Deploy with specific tag"
            echo "  IMAGE_TAG=v1.2.3 $0                  # Deploy via environment variable"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run main function
main "$@"