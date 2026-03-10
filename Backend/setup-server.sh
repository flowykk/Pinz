#!/bin/bash

# Complete Pinz Backend Server Setup for CI/CD
# This script sets up the entire server environment for automated deployment

set -e

# Configuration
REPO_URL="${REPO_URL:-https://github.com/flowykk/Pinz.git}"
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

# Check if running as root
check_root() {
    if [[ $EUID -eq 0 ]]; then
        log_error "This script should NOT be run as root"
        log_error "Create a regular user first:"
        log_error "  useradd -m -s /bin/bash deploy"
        log_error "  passwd deploy"
        log_error "  usermod -aG sudo deploy"
        log_error "Then switch to user: su - deploy"
        exit 1
    fi
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

# Install k3s (Kubernetes)
install_k3s() {
    log_info "Installing k3s..."
    curl -sfL https://get.k3s.io | sh -

    # Configure kubectl access for current user
    sudo chmod 644 /etc/rancher/k3s/k3s.yaml 2>/dev/null || true
    export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
    echo "export KUBECONFIG=/etc/rancher/k3s/k3s.yaml" >> ~/.bashrc

    # Wait for k3s to be ready
    sleep 10
    kubectl cluster-info

    # Ensure user can access Docker
    sudo usermod -aG docker $USER
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

# Install Helmfile
install_helmfile() {
    log_info "Installing Helmfile..."
    wget -O helmfile.tar.gz https://github.com/helmfile/helmfile/releases/latest/download/helmfile_1.3.1_linux_amd64.tar.gz
    tar -xzf helmfile.tar.gz
    sudo mv helmfile /usr/local/bin/
    rm helmfile.tar.gz

    helmfile version
    log_success "Helmfile installed"
}

# Install Istio
install_istio() {
    log_info "Installing istioctl..."
    curl -L https://istio.io/downloadIstioctl | sh -
    export PATH=$PATH:$HOME/.istioctl/bin
    echo "export PATH=\$PATH:\$HOME/.istioctl/bin" >> ~/.bashrc

    istioctl version
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

# Configure firewall
configure_firewall() {
    log_info "Configuring firewall..."
    sudo ufw allow 22/tcp    # SSH
    sudo ufw allow 80/tcp    # HTTP
    sudo ufw allow 443/tcp   # HTTPS
    sudo ufw allow 6443/tcp  # k3s API
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

    # PREROUTING handles external traffic to VPS.
    if ! sudo iptables -t nat -C PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "$http_nodeport" &>/dev/null; then
        sudo iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port "$http_nodeport"
    fi
    if ! sudo iptables -t nat -C PREROUTING -p tcp --dport 443 -j REDIRECT --to-port "$https_nodeport" &>/dev/null; then
        sudo iptables -t nat -A PREROUTING -p tcp --dport 443 -j REDIRECT --to-port "$https_nodeport"
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

# Clone repository
clone_repository() {
    log_info "Cloning repository..."
    if [[ -d "$PROJECT_DIR/.git" ]]; then
        log_warning "Repository already exists, updating..."
        cd $PROJECT_DIR
        git checkout develop
        git pull origin develop
    else
        cd $PROJECT_DIR
        git clone $REPO_URL .
        git checkout develop
        git pull origin develop
    fi

    # Ensure scripts are executable (if they exist)
    [[ -f "deploy.sh" ]] && chmod +x deploy.sh
    [[ -f "setup-server.sh" ]] && chmod +x setup-server.sh
    [[ -f "setup-certbot.sh" ]] && chmod +x setup-certbot.sh
    log_success "Repository cloned and switched to develop branch"
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

# Docker Registry (for production deployment)
DOCKER_REGISTRY=ghcr.io
DOCKER_REPO=flowykk/pinz

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

    # Start infrastructure
    make infra-up

    # Wait for infrastructure to be ready
    log_info "Waiting for infrastructure..."
    sleep 30

    # Check infrastructure status
    docker ps
    log_success "Infrastructure started"
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

# Main setup function
main() {
    log_info "🚀 Starting complete Pinz Backend server setup..."

    check_root
    update_system
    install_docker
    install_k3s
    install_helm
    install_helm_plugins
    install_helmfile
    install_istio
    install_tools
    configure_firewall
    setup_port_forwarding
    create_directories
    clone_repository
    create_env_file
    setup_infrastructure
    setup_istio
    setup_port_forwarding
    test_deployment
    create_deploy_user
    setup_ssh_for_cd
    print_instructions

    log_success "🎉 Setup completed! Your server is ready for CI/CD deployment."
}

# Handle command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --repo-url)
            REPO_URL="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Complete server setup for Pinz Backend CI/CD"
            echo ""
            echo "Options:"
            echo "  --repo-url URL     Repository URL to clone [default: $REPO_URL]"
            echo "  --help, -h         Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  REPO_URL           Same as --repo-url"
            echo "  POSTGRES_PASSWORD  Custom PostgreSQL password"
            echo "  JWT_SECRET_KEY     Custom JWT secret key"
            echo ""
            echo "Example:"
            echo "  $0 --repo-url https://github.com/your-org/pinz-backend.git"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run main setup
main "$@"