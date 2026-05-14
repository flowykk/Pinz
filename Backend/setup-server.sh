#!/bin/bash

# Complete Pinz Backend Server Setup for CI/CD
# This script sets up the entire server environment for automated deployment

set -e

# Configuration
REPO_URL="${REPO_URL:-https://github.com/flowykk/Pinz.git}"
BRANCH="${BRANCH:-develop}"
PROFILE="${PROFILE:-prod}"   # prod | loadtest
PROJECT_DIR="/opt/pinz"
BACKEND_DIR="${PROJECT_DIR}/Backend"
ENV_FILE="${BACKEND_DIR}/.env"

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

# Если запущено под root — создаём deploy и re-exec под ним.
bootstrap_deploy_from_root() {
    [[ $EUID -eq 0 ]] || return 0

    log_info "Running as root — bootstrapping deploy user..."
    id deploy &>/dev/null || useradd -m -s /bin/bash deploy
    usermod -aG sudo deploy
    echo "deploy ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/deploy
    chmod 440 /etc/sudoers.d/deploy

    if [[ -f /root/.ssh/authorized_keys ]]; then
        mkdir -p /home/deploy/.ssh
        cp /root/.ssh/authorized_keys /home/deploy/.ssh/authorized_keys
        chown -R deploy:deploy /home/deploy/.ssh
        chmod 700 /home/deploy/.ssh
        chmod 600 /home/deploy/.ssh/authorized_keys
    fi

    # /root mode 700, deploy не может читать $0. Копируем в /tmp.
    SCRIPT_PATH="$(readlink -f "$0")"
    local TMP_SCRIPT="/tmp/setup-server-deploy.sh"
    cp "$SCRIPT_PATH" "$TMP_SCRIPT"
    chown deploy:deploy "$TMP_SCRIPT"
    chmod 755 "$TMP_SCRIPT"
    exec sudo -u deploy -E -H bash "$TMP_SCRIPT" \
        --repo-url "$REPO_URL" --branch "$BRANCH" --profile "$PROFILE"
}

# Update system
update_system() {
    log_info "Updating system packages..."
    sudo apt update && sudo apt upgrade -y
    log_success "System updated"
}

# Install Docker
install_docker() {
    log_info "Installing Docker..."
    curl -fsSL https://get.docker.com | sh
    sudo usermod -aG docker $USER

    # Start and enable Docker
    sudo systemctl enable docker
    sudo systemctl start docker

    # Test Docker
    docker --version
    log_success "Docker installed"
}

install_k3s() {
    log_info "Installing k3s..."
    curl -sfL https://get.k3s.io | sh -

    mkdir -p "$HOME/.kube"
    sudo cp /etc/rancher/k3s/k3s.yaml "$HOME/.kube/config"
    sudo chown "$USER:$USER" "$HOME/.kube/config"
    chmod 600 "$HOME/.kube/config"
    grep -q 'KUBECONFIG=' "$HOME/.bashrc" || echo 'export KUBECONFIG=$HOME/.kube/config' >> "$HOME/.bashrc"

    # /etc/profile.d — login shells (включая `ssh host`) подхватывают env.
    sudo tee /etc/profile.d/pinz.sh > /dev/null <<EOF
export KUBECONFIG=$HOME/.kube/config
export PATH=\$PATH:/usr/local/go/bin
EOF
    sudo chmod 644 /etc/profile.d/pinz.sh

    until sudo kubectl get nodes &>/dev/null; do sleep 2; done

    sudo usermod -aG docker "$USER"
    log_success "k3s installed"
}

# Install Helm
install_helm() {
    log_info "Installing Helm..."
    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    helm version --short
    log_success "Helm installed"
}

# Install Helm plugins
install_helm_plugins() {
    log_info "Installing Helm plugins..."

    # Install helm-diff plugin
    log_info "Installing helm-diff plugin..."
    helm plugin install https://github.com/databus23/helm-diff --version v3.14.0
    if [[ $? -ne 0 ]]; then
        log_error "Failed to install helm-diff plugin"
        exit 1
    fi

    # Verify helm-diff is working
    if ! helm diff version &> /dev/null; then
        log_error "helm-diff plugin is installed but not working"
        exit 1
    fi

    log_success "Helm plugins installed"
}

# Hardcoded URL 404'ит после каждого нового тега, поэтому резолвим latest динамически.
install_helmfile() {
    log_info "Installing Helmfile..."
    local ver
    ver=$(curl -sIL https://github.com/helmfile/helmfile/releases/latest \
        | grep -i ^location: | tail -1 | sed -E 's|.*/v([0-9.]+).*|\1|' | tr -d '\r')
    if [[ -z "$ver" ]]; then
        log_error "Cannot resolve helmfile latest version"
        exit 1
    fi
    local tmp
    tmp=$(mktemp -d)
    wget -q -O "$tmp/h.tgz" "https://github.com/helmfile/helmfile/releases/download/v${ver}/helmfile_${ver}_linux_amd64.tar.gz"
    tar -xzf "$tmp/h.tgz" -C "$tmp"
    sudo mv "$tmp/helmfile" /usr/local/bin/
    rm -rf "$tmp"

    helmfile version
    log_success "Helmfile installed (v$ver)"
}

# Symlink — чтобы istioctl был виден в неинтерактивных ssh-вызовах (.bashrc не source-ится).
install_istio() {
    log_info "Installing istioctl..."
    curl -fsSL https://istio.io/downloadIstioctl | sh -
    export PATH=$PATH:$HOME/.istioctl/bin
    grep -q '.istioctl/bin' ~/.bashrc || echo "export PATH=\$PATH:\$HOME/.istioctl/bin" >> ~/.bashrc
    sudo ln -sf "$HOME/.istioctl/bin/istioctl" /usr/local/bin/istioctl
    istioctl version --remote=false
    log_success "istioctl installed"
}

# Install additional tools
install_tools() {
    log_info "Installing additional tools..."
    sudo apt install -y git make protobuf-compiler jq htop curl wget

    # Install Go (for building)
    wget -O go.tar.gz https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo "export PATH=\$PATH:/usr/local/go/bin" >> ~/.bashrc
    rm go.tar.gz

    go version
    log_success "Additional tools installed"
}

configure_firewall() {
    log_info "Configuring firewall (profile=$PROFILE)..."
    sudo ufw allow 22/tcp    # SSH
    sudo ufw allow 80/tcp    # HTTP
    sudo ufw allow 443/tcp   # HTTPS
    sudo ufw allow 6443/tcp  # k3s API
    if [[ "$PROFILE" == "loadtest" ]]; then
        sudo ufw allow 30000:32767/tcp comment "k8s NodePort (loadtest)"
        sudo ufw allow 8080/tcp comment "API direct (loadtest)"
    fi
    sudo ufw --force enable
    log_success "Firewall configured"
}

# Forward ports 80/443 to Istio IngressGateway NodePorts.
# k3s does not have a cloud LoadBalancer by default, so we resolve real NodePort
# values from istio-ingressgateway and map standard ports via iptables.
setup_port_forwarding() {
    local http_nodeport
    local https_nodeport
    http_nodeport=$(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}')
    https_nodeport=$(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath='{.spec.ports[?(@.port==443)].nodePort}')

    if [[ -z "$http_nodeport" ]] || [[ -z "$https_nodeport" ]]; then
        log_warning "Cannot resolve Istio ingress NodePorts, skipping iptables forwarding"
        return 0
    fi

    log_info "Setting up port forwarding 80→${http_nodeport}, 443→${https_nodeport}..."

    # Remove legacy broad redirects first. They also catch forwarded pod egress
    # to the internet on ports 80/443, which breaks outbound connectivity.
    sudo iptables -t nat -D PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "$http_nodeport" 2>/dev/null || true
    sudo iptables -t nat -D PREROUTING -p tcp --dport 443 -j REDIRECT --to-port "$https_nodeport" 2>/dev/null || true

    # PREROUTING handles external traffic to VPS. Match only node-local
    # destinations so pod egress to arbitrary remote 80/443 is not intercepted.
    if ! sudo iptables -t nat -C PREROUTING -m addrtype --dst-type LOCAL -p tcp --dport 80 -j REDIRECT --to-port "$http_nodeport" &>/dev/null; then
        sudo iptables -t nat -A PREROUTING -m addrtype --dst-type LOCAL -p tcp --dport 80 -j REDIRECT --to-port "$http_nodeport"
    fi
    if ! sudo iptables -t nat -C PREROUTING -m addrtype --dst-type LOCAL -p tcp --dport 443 -j REDIRECT --to-port "$https_nodeport" &>/dev/null; then
        sudo iptables -t nat -A PREROUTING -m addrtype --dst-type LOCAL -p tcp --dport 443 -j REDIRECT --to-port "$https_nodeport"
    fi

    # OUTPUT handles local checks like curl http://localhost/health.
    if ! sudo iptables -t nat -C OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "$http_nodeport" &>/dev/null; then
        sudo iptables -t nat -A OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-port "$http_nodeport"
    fi
    if ! sudo iptables -t nat -C OUTPUT -p tcp -d 127.0.0.1 --dport 443 -j REDIRECT --to-port "$https_nodeport" &>/dev/null; then
        sudo iptables -t nat -A OUTPUT -p tcp -d 127.0.0.1 --dport 443 -j REDIRECT --to-port "$https_nodeport"
    fi

    # Persist rules across reboots
    if command -v netfilter-persistent &>/dev/null; then
        sudo netfilter-persistent save
    else
        sudo apt-get install -y iptables-persistent
        sudo netfilter-persistent save
    fi

    log_success "Port forwarding configured: :80 → NodePort ${http_nodeport}, :443 → NodePort ${https_nodeport}"
}

# Create project directories
create_directories() {
    log_info "Creating project directories..."
    sudo mkdir -p $PROJECT_DIR
    sudo chown $USER:$USER $PROJECT_DIR
    log_success "Directories created"
}

clone_repository() {
    log_info "Cloning repository (branch=$BRANCH)..."
    if [[ -d "$PROJECT_DIR/.git" ]]; then
        log_warning "Repository already exists, updating..."
        cd "$PROJECT_DIR"
        git fetch --all
        git checkout "$BRANCH"
        git pull origin "$BRANCH"
    else
        cd "$PROJECT_DIR"
        git clone "$REPO_URL" .
        git fetch --all
        git checkout "$BRANCH"
        git pull origin "$BRANCH"
    fi

    [[ -f "deploy.sh" ]] && chmod +x deploy.sh
    [[ -f "setup-cert-manager.sh" ]] && chmod +x setup-cert-manager.sh
    [[ -f "setup-server.sh" ]] && chmod +x setup-server.sh
    [[ -f "setup-certbot.sh" ]] && chmod +x setup-certbot.sh
    log_success "Repository cloned at branch '$BRANCH'"
}

# Create environment file
create_env_file() {
    log_info "Creating environment configuration..."

    # Determine server IP
    local server_ip
    server_ip=$(ip route get 1 2>/dev/null | awk '{print $7}' || hostname -I | awk '{print $1}' || echo "host.docker.internal")

    if [[ ! -f "$ENV_FILE" ]]; then
        cat > "$ENV_FILE" << EOF
# Pinz Backend Environment Configuration
# Generated by setup-server.sh

# Server Configuration
SERVER_IP=${server_ip}
DOMAIN=${DOMAIN:-pinz.website}
# k3s pulls images via containerd directly; docker pull is not needed on this server.
SKIP_PULL=true

# Database
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-pinz_secure_password_$(openssl rand -hex 16)}
DB_HOST=db
DB_PORT=5432
DB_USER=pinz_user
DB_NAME=pinz_db

# Redis
REDIS_ADDR=redis:6379

# JWT
JWT_SECRET_KEY=${JWT_SECRET_KEY:-pinz_jwt_secret_$(openssl rand -hex 32)}

# Application
API_GATEWAY_PORT=8080
GRPC_PORT=:50051

# Docker Registry (loadtest pulls public images, prod — private)
DOCKER_REGISTRY=ghcr.io
DOCKER_REPO=$(if [[ "$PROFILE" == "loadtest" ]]; then echo "dmitry-pr"; else echo "flowykk/pinz"; fi)

# Environment
ENVIRONMENT=prod
EOF
        log_success "Environment file created: $ENV_FILE"
        log_warning "Please review and update the generated passwords in $ENV_FILE"
    else
        log_warning "Environment file already exists: $ENV_FILE"
    fi
}

# Setup infrastructure
setup_infrastructure() {
    log_info "Setting up infrastructure..."

    cd $BACKEND_DIR

    # Load environment variables
    if [[ -f "$ENV_FILE" ]]; then
        set -a
        source "$ENV_FILE"
        set +a
    fi

    # Check if Makefile exists and has infra-up target
    if [[ ! -f "Makefile" ]]; then
        log_error "Makefile not found in $BACKEND_DIR"
        log_error "Please ensure you're using the correct branch with deployment files"
        exit 1
    fi

    if ! grep -q "^infra-up:" Makefile; then
        log_error "infra-up target not found in Makefile"
        log_error "Available targets:"
        make -n 2>/dev/null | head -10 || echo "No make targets available"
        exit 1
    fi

    # usermod -aG docker применяется к новым shell'ам; в текущем — нет, sg
    # подхватывает группу для одной команды.
    sg docker -c "make infra-up"

    # Wait for infrastructure to be ready
    log_info "Waiting for infrastructure..."
    sleep 30

    # Check infrastructure status
    docker ps
    log_success "Infrastructure started"
}

# Kiali + sample Prometheus addons pinned to running Istio minor (release branch).
setup_kiali_addons() {
    log_info "Installing Kiali + Prometheus addons..."
    local ver minor base
    ver=$(istioctl version --remote=false 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    if [[ -z "$ver" ]]; then
        log_error "Cannot resolve installed istio version"
        exit 1
    fi
    minor="${ver%.*}"
    base="https://raw.githubusercontent.com/istio/istio/release-${minor}/samples/addons"
    kubectl apply -f "${base}/prometheus.yaml"
    kubectl apply -f "${base}/kiali.yaml"
    kubectl wait --for=condition=available --timeout=300s -n istio-system \
        deployment/prometheus deployment/kiali
    log_success "Kiali addons installed (istio release-${minor})"
}

# Setup Istio
setup_istio() {
    log_info "Setting up Istio..."

    cd $BACKEND_DIR

    # Install Istio profile
    istioctl install --set profile=default -y

    # Enable sidecar injection
    kubectl label namespace default istio-injection=enabled

    # Wait for Istio to be ready
    kubectl wait --for=condition=available --timeout=300s deployment/istiod -n istio-system

    log_success "Istio configured"
}

# Test deployment
test_deployment() {
    log_info "Testing deployment..."

    cd $BACKEND_DIR

    # Load environment
    if [[ -f "$ENV_FILE" ]]; then
        set -a
        source "$ENV_FILE"
        set +a
    fi

    # Test deploy script (if exists)
    if [[ -f "deploy.sh" ]]; then
        ./deploy.sh --help
    else
        log_warning "deploy.sh not found - skipping deployment test"
    fi

    # Check kubectl access
    kubectl get nodes
    kubectl get pods

    log_success "Deployment test passed"
}

# Create CI/CD user (optional)
create_deploy_user() {
    log_info "Setting up deploy user for CI/CD..."

    # Create deploy user if it doesn't exist
    if ! id "deploy" &>/dev/null; then
        sudo useradd -m -s /bin/bash deploy
        sudo usermod -aG docker deploy

        # Create SSH directory
        sudo mkdir -p /home/deploy/.ssh
        sudo chown deploy:deploy /home/deploy/.ssh
        sudo chmod 700 /home/deploy/.ssh

        log_success "Deploy user created"
    else
        log_warning "Deploy user already exists"
    fi

    # Setup sudo for deploy user
    echo "deploy ALL=(ALL) NOPASSWD: ALL" | sudo tee /etc/sudoers.d/deploy > /dev/null
}

# Setup SSH for CI/CD
setup_ssh_for_cd() {
    log_info "Setting up SSH access for CI/CD..."

    # Generate SSH key for CI/CD if it doesn't exist
    SSH_KEY_PATH="$HOME/.ssh/pinz_cd_key"
    if [[ ! -f "$SSH_KEY_PATH" ]]; then
        ssh-keygen -t ed25519 -C "pinz-cd@server" -f "$SSH_KEY_PATH" -N ""

        log_success "SSH key generated for CI/CD"
        echo ""
        echo "=== CI/CD SSH Setup ==="
        echo "Add this public key to your GitHub repository deploy keys:"
        echo "GitHub → Repository → Settings → Deploy keys → Add deploy key"
        echo ""
        cat "${SSH_KEY_PATH}.pub"
        echo ""
        echo "And add this private key to GitHub secrets as VPS_SSH_KEY:"
        echo "GitHub → Repository → Settings → Secrets and variables → Actions"
        echo ""
        cat "$SSH_KEY_PATH"
        echo ""
        echo "======================="
    else
        log_warning "SSH key already exists: $SSH_KEY_PATH"
    fi
}

# Print final instructions
print_instructions() {
    echo ""
    log_success "🎉 Server setup completed successfully!"
    echo ""
    echo "=== Next Steps ==="
    echo ""
    echo "1. Review and update passwords in: $ENV_FILE"
    echo ""
    echo "2. For CI/CD, add these GitHub secrets:"
    echo "   - VPS_HOST: $(hostname -I | awk '{print $1}')"
    echo "   - VPS_USER: $USER"
    echo "   - VPS_SSH_KEY: (see above)"
    echo "   - POSTGRES_PASSWORD: (from $ENV_FILE)"
    echo "   - JWT_SECRET_KEY: (from $ENV_FILE)"
    echo ""
    echo "3. Test manual deployment:"
    echo "   cd $BACKEND_DIR"
    echo "   ./deploy.sh --environment prod"
    echo ""
    echo "4. Available make commands:"
    echo "   - make infra-up      # Start infrastructure"
    echo "   - make infra-down    # Stop infrastructure"
    echo "   - make k8s-deploy    # Deploy to Kubernetes"
    echo "   - make k8s-build     # Build Docker images"
    echo "   - make proto         # Generate protobuf files"
    echo "   - make swagger       # Generate swagger docs"
    echo ""
    echo "5. Check status:"
    echo "   kubectl get pods"
    echo "   curl http://localhost:8080/health"
    echo ""
    echo "6. View logs:"
    echo "   kubectl logs -f deployment/api-gateway"
    echo "   kubectl logs -f deployment/auth-service"
    echo ""
}

# Guard: abort if the server is already set up unless --force is passed.
# Re-running setup reinstalls k3s and Istio which breaks running deployments.
check_already_setup() {
    local already=0

    command -v kubectl &>/dev/null && kubectl get nodes &>/dev/null && already=1

    if [[ "$already" -eq 1 ]] && [[ "${FORCE_SETUP:-false}" != "true" ]]; then
        log_error "Server appears to already be set up (kubectl cluster is accessible)."
        log_error ""
        log_error "Re-running setup-server.sh will reinstall k3s and Istio, breaking"
        log_error "any running deployments."
        log_error ""
        log_error "If you only want to deploy the application, use:"
        log_error "  cd /opt/pinz/Backend && ./deploy.sh"
        log_error ""
        log_error "To force a full re-setup anyway (DANGEROUS):"
        log_error "  FORCE_SETUP=true ./setup-server.sh"
        exit 1
    fi
}

# prod — полный сценарий с deploy.sh; loadtest — без test_deployment
# (стенд деплоится отдельным `helmfile -f helmfile.loadtest.yaml.gotmpl sync`).
main() {
    log_info "🚀 Starting Pinz Backend server setup (profile=$PROFILE, branch=$BRANCH)..."

    bootstrap_deploy_from_root
    check_already_setup
    update_system
    install_docker
    install_k3s
    install_helm
    install_helm_plugins
    install_helmfile
    install_istio
    install_tools
    configure_firewall
    create_directories
    clone_repository
    create_env_file
    setup_infrastructure
    setup_istio
    setup_kiali_addons
    setup_port_forwarding
    create_deploy_user
    setup_ssh_for_cd

    if [[ "$PROFILE" == "prod" ]]; then
        test_deployment
    else
        log_info "Profile=loadtest — skipping prod test_deployment. Run loadtest deploy with:"
        log_info "  cd $BACKEND_DIR && helmfile -f helmfile.loadtest.yaml.gotmpl sync"
    fi

    print_instructions

    log_success "🎉 Setup completed! profile=$PROFILE"
}

# Handle command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --repo-url)
            REPO_URL="$2"; shift 2 ;;
        --branch)
            BRANCH="$2"; shift 2 ;;
        --profile)
            PROFILE="$2"; shift 2
            if [[ "$PROFILE" != "prod" && "$PROFILE" != "loadtest" ]]; then
                log_error "--profile must be 'prod' or 'loadtest', got '$PROFILE'"
                exit 1
            fi
            ;;
        --help|-h)
            cat <<EOF
Usage: $0 [OPTIONS]

Полная установка Pinz Backend на чистом VPS.

Options:
  --repo-url URL     Repository URL to clone (default: $REPO_URL)
  --branch BRANCH    Git branch (default: develop). Для loadtest-стенда из
                     feature-ветки: --branch PINZ-NNN
  --profile P        prod | loadtest (default: prod). Loadtest пропускает
                     test_deployment и открывает 8080/30000-32767 в ufw.
  --help, -h         Show this help

Environment variables (equivalent to flags):
  REPO_URL           Repository URL
  BRANCH             Git branch
  PROFILE            prod | loadtest
  POSTGRES_PASSWORD  Custom PostgreSQL password
  JWT_SECRET_KEY     Custom JWT secret key

Examples:
  # Прод-стенд из develop
  sudo $0
  # Loadtest-стенд из ветки PINZ-206
  sudo $0 --profile loadtest --branch PINZ-206
EOF
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"; exit 1 ;;
    esac
done

# Run main setup
main "$@"