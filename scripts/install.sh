#!/bin/bash
set -e

# Configuration
AGENT_USER="bpf-agent"
BINARY_NAME="bpfstack-agent"
INSTALL_DIR="/usr/local/bin"
SYSTEMD_DIR="/etc/systemd/system"

# Resolve script directory for relative paths
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
# Assuming the script is in scripts/ relative to the root, or just in the root.
# Let's try to find the root of the repo.
if [ -f "$SCRIPT_DIR/../go.mod" ]; then
    PROJECT_ROOT="$SCRIPT_DIR/.."
elif [ -f "$SCRIPT_DIR/go.mod" ]; then
    PROJECT_ROOT="$SCRIPT_DIR"
else
    echo "Error: Could not locate project root (go.mod not found)."
    exit 1
fi

SERVICE_FILE="$PROJECT_ROOT/deploy/bpfstack-agent.service"

echo "=== BPFStack Agent Installer ==="
echo "Script Dir: $SCRIPT_DIR"
echo "Project Root: $PROJECT_ROOT"

# 1. Check for root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root (sudo ./install.sh)"
  exit 1
fi

# 2. Create Service User
echo "[+] Creating service user: $AGENT_USER"
if ! id "$AGENT_USER" &>/dev/null; then
    useradd -r -s /bin/false "$AGENT_USER"
    echo "    User created."
else
    echo "    User already exists."
fi

# 3. Build Agent
echo "[+] Building agent..."
pushd "$PROJECT_ROOT" > /dev/null

if command -v go &>/dev/null; then
    go build -o agent ./cmd/agent
elif [ -f "/usr/local/go/bin/go" ]; then
    /usr/local/go/bin/go build -o agent ./cmd/agent
else
    if [ ! -f "./agent" ]; then
        echo "Error: 'go' not found and no './agent' binary present. Cannot build."
        popd > /dev/null
        exit 1
    fi
    echo "    'go' not found, using existing './agent' binary."
fi

# 4. Install Binary
echo "[+] Installing binary to $INSTALL_DIR/$BINARY_NAME"
cp ./agent "$INSTALL_DIR/$BINARY_NAME"
chmod 755 "$INSTALL_DIR/$BINARY_NAME"
popd > /dev/null

# 5. Install Systemd Service
echo "[+] Installing systemd service..."
if [ ! -f "$SERVICE_FILE" ]; then
    echo "Error: Service file $SERVICE_FILE not found."
    exit 1
fi
cp "$SERVICE_FILE" "$SYSTEMD_DIR/"
systemctl daemon-reload

# 6. Enable and Start Service
echo "[+] Enabling and starting service..."
systemctl enable bpfstack-agent
systemctl restart bpfstack-agent

echo "=== Installation Complete ==="
echo "Check status with: systemctl status bpfstack-agent"
