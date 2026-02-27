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

# Update repository
update_repo() {
    if is_server && [[ -d ".git" ]]; then
        log_info "Updating repository on server..."

        # Get current branch
        local current_branch=$(git branch --show-current)

        # Pull latest changes
        git pull origin "$current_branch"

        # Reset any local changes (optional, be careful with this)
        # git reset --hard origin/"$current_branch"

        log_success "Repository updated successfully"
    fi
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
    POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-pinz_password}"
    JWT_SECRET_KEY="${JWT_SECRET_KEY:-change-me-in-production}"
}

# Validate environment
validate_env() {
    log_info "Validating environment..."

    # Check required tools
    local required_tools=("docker" "kubectl" "helm" "helmfile")

    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "$tool is not installed or not in PATH"
            exit 1
        fi
    done

    # Check Docker daemon
    if ! docker info &> /dev/null; then
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

# Setup Docker registry authentication
setup_docker_auth() {
    if is_ci; then
        log_info "Setting up Docker authentication for CI"
        echo "${GITHUB_TOKEN:-}" | docker login "$DOCKER_REGISTRY" -u "${GITHUB_ACTOR:-}" --password-stdin
    else
        log_info "Setting up Docker authentication for server"
        # Try to login with GitHub token if available
        if [[ -n "${GITHUB_TOKEN:-}" ]]; then
            echo "$GITHUB_TOKEN" | docker login "$DOCKER_REGISTRY" -u "${GITHUB_ACTOR:-$USER}" --password-stdin
        else
            log_warning "GITHUB_TOKEN not set. Please login to Docker registry manually:"
            log_warning "docker login $DOCKER_REGISTRY"
            log_warning "Or set GITHUB_TOKEN environment variable"
            # Try to login anyway, maybe user already logged in
            if ! docker pull "${DOCKER_REGISTRY}/${DOCKER_REPO}/pinz-api-gateway:${IMAGE_TAG}" &>/dev/null; then
                log_error "Cannot access Docker registry. Please authenticate first."
                exit 1
            fi
        fi
    fi
}

# Pull Docker images
pull_images() {
    local api_gateway_image="${DOCKER_REGISTRY}/${DOCKER_REPO}/pinz-api-gateway:${IMAGE_TAG}"
    local auth_service_image="${DOCKER_REGISTRY}/${DOCKER_REPO}/pinz-auth-service:${IMAGE_TAG}"

    log_info "Pulling Docker images..."
    log_info "API Gateway: $api_gateway_image"
    log_info "Auth Service: $auth_service_image"

    docker pull "$api_gateway_image"
    docker pull "$auth_service_image"

    log_success "Images pulled successfully"
}

# Select Helmfile configuration
select_helmfile() {
    HELMFILE_CONFIG="${PROJECT_DIR}/helmfile.yaml.gotmpl"

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
    export DOCKER_REGISTRY="$DOCKER_REGISTRY"
    export DOCKER_REPO="$DOCKER_REPO"
    export IMAGE_TAG="$IMAGE_TAG"
    export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-pinz_password}"
    export JWT_SECRET_KEY="${JWT_SECRET_KEY:-change-me-in-production}"

    cd "$PROJECT_DIR"
    helmfile -f "$HELMFILE_CONFIG" apply

    log_success "Application deployed successfully"
}

# Wait for deployment to be ready
wait_for_deployment() {
    log_info "Waiting for deployment to be ready..."

    # Wait for pods to be ready
    kubectl wait --for=condition=available --timeout=300s deployment/api-gateway
    kubectl wait --for=condition=available --timeout=300s deployment/auth-service

    # Wait for rollout to complete
    kubectl rollout status deployment/api-gateway --timeout=300s
    kubectl rollout status deployment/auth-service --timeout=300s

    log_success "Deployment is ready"
}

# Health check
health_check() {
    log_info "Performing health checks..."

    # Get service URL
    local api_url=""
    if is_server; then
        # On server, check internal service
        api_url="http://api-gateway.default.svc.cluster.local:8080"
    else
        # Locally, try to get from minikube or ingress
        api_url="http://localhost:8080"
    fi

    # Wait for service to respond
    local max_attempts=30
    local attempt=1

    while [[ $attempt -le $max_attempts ]]; do
        if curl -f -s "$api_url/health" &> /dev/null; then
            log_success "Health check passed: $api_url/health"
            return 0
        fi

        log_info "Health check attempt $attempt/$max_attempts failed, retrying..."
        sleep 10
        ((attempt++))
    done

    log_error "Health check failed after $max_attempts attempts"
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

    # Update repository if on server
    update_repo

    # Load environment
    load_env

    # Validate environment
    validate_env

    # Setup Docker auth
    setup_docker_auth

    # Pull images
    pull_images

    # Deploy application
    deploy_app

    # Wait for readiness
    wait_for_deployment

    # Health check
    health_check

    # Show status
    show_status

    log_success "🎉 Deployment completed successfully!"
    log_info "Application is available at: http://<your-server-ip>:8080"
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
            echo "  POSTGRES_PASSWORD        Database password"
            echo "  JWT_SECRET_KEY           JWT secret key"
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