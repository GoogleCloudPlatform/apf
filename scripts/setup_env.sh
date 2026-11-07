#!/bin/bash
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

echo "================================================================="
echo "Initializing Centralized Environment Setup (Mise Orchestration)"
echo "================================================================="

ROOT_DIR=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
CONFIG_FILE="$ROOT_DIR/config.env"
TEMPLATE_FILE="$ROOT_DIR/config.env.template"

if [ ! -f "$CONFIG_FILE" ]; then
    if [ -f "$TEMPLATE_FILE" ]; then
        echo "Creating local config.env from config.env.template..."
        cp "$TEMPLATE_FILE" "$CONFIG_FILE"
    else
        echo "Error: Neither config.env nor config.env.template found."
        exit 1
    fi
fi

echo "Sourcing master configuration from $CONFIG_FILE..."
source "$CONFIG_FILE"

if ! grep -q "^GOOGLE_CLOUD_PROJECT=" "$CONFIG_FILE"; then
    echo "GOOGLE_CLOUD_PROJECT=" >> "$CONFIG_FILE"
fi

if [ -z "$GOOGLE_CLOUD_PROJECT" ] || [ "$GOOGLE_CLOUD_PROJECT" = "your-gcp-project-id" ]; then
    echo "================================================================="
    echo "☁️  Google Cloud Setup: GOOGLE_CLOUD_PROJECT is not initialized."
    echo "Please enter your Google Cloud Project ID (it will be saved to config.env):"
    read -p "Project ID: " USER_GCP_PROJECT
    echo ""
    if [ -n "$USER_GCP_PROJECT" ]; then
        sed -i "s|^GOOGLE_CLOUD_PROJECT=.*|GOOGLE_CLOUD_PROJECT=$USER_GCP_PROJECT|" "$CONFIG_FILE"
        export GOOGLE_CLOUD_PROJECT="$USER_GCP_PROJECT"
        echo "✅ Google Cloud Project ID saved to config.env."
    else
        echo "⚠️ No Project ID entered. Cloud operations may fail."
    fi
    echo "================================================================="
fi

if ! grep -q "^GOOGLE_CLOUD_LOCATION=" "$CONFIG_FILE"; then
    echo "GOOGLE_CLOUD_LOCATION=" >> "$CONFIG_FILE"
fi
if [ -z "$GOOGLE_CLOUD_LOCATION" ]; then
    echo "================================================================="
    echo "🌍 Google Cloud Setup: GOOGLE_CLOUD_LOCATION is not initialized."
    read -p "Location [default: us-central1]: " USER_GCP_LOCATION
    USER_GCP_LOCATION="${USER_GCP_LOCATION:-us-central1}"
    sed -i "s|^GOOGLE_CLOUD_LOCATION=.*|GOOGLE_CLOUD_LOCATION=$USER_GCP_LOCATION|" "$CONFIG_FILE"
    export GOOGLE_CLOUD_LOCATION="$USER_GCP_LOCATION"
    echo "✅ Google Cloud Location saved to config.env: $GOOGLE_CLOUD_LOCATION"
    echo "================================================================="
fi

if ! grep -q "^GCS_ARTIFACT_BUCKET=" "$CONFIG_FILE"; then
    echo "GCS_ARTIFACT_BUCKET=" >> "$CONFIG_FILE"
fi
if [ -z "$GCS_ARTIFACT_BUCKET" ] || [ "$GCS_ARTIFACT_BUCKET" = "your-artifact-bucket" ]; then
    echo "================================================================="
    echo "🪣 Google Cloud Storage Setup: GCS_ARTIFACT_BUCKET is not initialized."
    echo "Please enter your GCS Artifact Bucket name without the leading gs:// (it will be saved to config.env):"
    read -p "Artifact Bucket: " USER_ARTIFACT_BUCKET
    USER_ARTIFACT_BUCKET="${USER_ARTIFACT_BUCKET#gs://}"
    echo ""
    if [ -n "$USER_ARTIFACT_BUCKET" ]; then
        sed -i "s|^GCS_ARTIFACT_BUCKET=.*|GCS_ARTIFACT_BUCKET=$USER_ARTIFACT_BUCKET|" "$CONFIG_FILE"
        export GCS_ARTIFACT_BUCKET="$USER_ARTIFACT_BUCKET"
        echo "✅ GCS Artifact Bucket saved to config.env."
    else
        echo "⚠️ No Artifact Bucket entered. Remote artifact storage may fail."
    fi
    echo "================================================================="
fi

if ! grep -q "^WORKSPACE_MOUNT_PATH=" "$CONFIG_FILE"; then
    echo "WORKSPACE_MOUNT_PATH=" >> "$CONFIG_FILE"
fi
DEFAULT_MOUNT_PATH="$HOME/.apf"
if [ -z "$WORKSPACE_MOUNT_PATH" ] || [[ "$WORKSPACE_MOUNT_PATH" == \~* ]]; then
    echo "================================================================="
    echo "📁 Workspace Mount Path Setup: WORKSPACE_MOUNT_PATH is not initialized."
    read -p "Workspace Mount Path [default: $DEFAULT_MOUNT_PATH]: " USER_MOUNT_PATH
    USER_MOUNT_PATH="${USER_MOUNT_PATH:-$DEFAULT_MOUNT_PATH}"
    USER_MOUNT_PATH="${USER_MOUNT_PATH/#\~/$HOME}"
    sed -i "s|^WORKSPACE_MOUNT_PATH=.*|WORKSPACE_MOUNT_PATH=$USER_MOUNT_PATH|" "$CONFIG_FILE"
    export WORKSPACE_MOUNT_PATH="$USER_MOUNT_PATH"
    echo "✅ Workspace Mount Path saved to config.env: $WORKSPACE_MOUNT_PATH"
    echo "================================================================="
fi

if ! grep -q "^LOCAL_ARTIFACT_PATH=" "$CONFIG_FILE"; then
    echo "LOCAL_ARTIFACT_PATH=.foundry/artifacts" >> "$CONFIG_FILE"
fi
LOCAL_ARTIFACT_PATH="${LOCAL_ARTIFACT_PATH:-.foundry/artifacts}"
export LOCAL_ARTIFACT_PATH

if ! grep -q "^BMAD_ROOT_PATH=" "$CONFIG_FILE"; then
    echo "BMAD_ROOT_PATH=" >> "$CONFIG_FILE"
fi
DEFAULT_BMAD_PATH="$HOME/.apf/config/bmad"
if [ -z "$BMAD_ROOT_PATH" ] || [[ "$BMAD_ROOT_PATH" == \~* ]]; then
    echo "================================================================="
    echo "🧠 BMAD Configuration Setup: BMAD_ROOT_PATH is not initialized."
    read -p "BMAD Root Path [default: $DEFAULT_BMAD_PATH]: " USER_BMAD_PATH
    USER_BMAD_PATH="${USER_BMAD_PATH:-$DEFAULT_BMAD_PATH}"
    USER_BMAD_PATH="${USER_BMAD_PATH/#\~/$HOME}"
    sed -i "s|^BMAD_ROOT_PATH=.*|BMAD_ROOT_PATH=$USER_BMAD_PATH|" "$CONFIG_FILE"
    export BMAD_ROOT_PATH="$USER_BMAD_PATH"
    echo "✅ BMAD Root Path saved to config.env: $BMAD_ROOT_PATH"
    echo "================================================================="
fi

if [ -z "$GEMINI_API_KEY" ] || [ "$GEMINI_API_KEY" = "your-api-key-here" ]; then
    echo "================================================================="
    echo "🔒 Security Setup: GEMINI_API_KEY is required."
    echo "Please enter your Gemini API Key (it will be saved securely in config.env):"
    read -s -p "API Key: " USER_API_KEY
    echo ""
    if [ -n "$USER_API_KEY" ]; then
        sed -i "s/GEMINI_API_KEY=.*/GEMINI_API_KEY=$USER_API_KEY/" "$CONFIG_FILE"
        export GEMINI_API_KEY="$USER_API_KEY"
        echo "✅ API Key saved to config.env."
    else
        echo "⚠️ No API Key entered. AI services may fail to authenticate."
    fi
    echo "================================================================="
fi

if [ -z "$DEV_REPO_ACCESS_TOKEN" ] && [ -n "$GITHUB_ACCESS_TOKEN" ]; then
    DEV_REPO_ACCESS_TOKEN="$GITHUB_ACCESS_TOKEN"
    sed -i "s/DEV_REPO_ACCESS_TOKEN=.*/DEV_REPO_ACCESS_TOKEN=$DEV_REPO_ACCESS_TOKEN/" "$CONFIG_FILE"
elif [ -z "$GITHUB_ACCESS_TOKEN" ] && [ -n "$DEV_REPO_ACCESS_TOKEN" ]; then
    GITHUB_ACCESS_TOKEN="$DEV_REPO_ACCESS_TOKEN"
    sed -i "s/GITHUB_ACCESS_TOKEN=.*/GITHUB_ACCESS_TOKEN=$GITHUB_ACCESS_TOKEN/" "$CONFIG_FILE"
fi

if [ -z "$DEV_REPO_ACCESS_TOKEN" ] || [ "$DEV_REPO_ACCESS_TOKEN" = "your-token-here" ]; then
    echo "================================================================="
    echo "🔒 Security Setup: DEV_REPO_ACCESS_TOKEN / GITHUB_ACCESS_TOKEN is required for accessing GitHub repositories."
    echo "Please enter your GitHub Access Token (it will be saved securely in config.env):"
    read -s -p "Access Token: " USER_REPO_TOKEN
    echo ""
    if [ -n "$USER_REPO_TOKEN" ]; then
        sed -i "s/DEV_REPO_ACCESS_TOKEN=.*/DEV_REPO_ACCESS_TOKEN=$USER_REPO_TOKEN/" "$CONFIG_FILE"
        sed -i "s/GITHUB_ACCESS_TOKEN=.*/GITHUB_ACCESS_TOKEN=$USER_REPO_TOKEN/" "$CONFIG_FILE"
        export DEV_REPO_ACCESS_TOKEN="$USER_REPO_TOKEN"
        export GITHUB_ACCESS_TOKEN="$USER_REPO_TOKEN"
        echo "✅ GitHub Access Token saved to config.env."
    else
        echo "⚠️ No Access Token entered. Repository access may fail."
    fi
    echo "================================================================="
fi

if ! grep -q "^SECRETS_ENCRYPTION_KEY=" "$CONFIG_FILE"; then
    echo "SECRETS_ENCRYPTION_KEY=dev-32-bytes-secrets-key-apfninj" >> "$CONFIG_FILE"
fi
if [ -z "$SECRETS_ENCRYPTION_KEY" ]; then
    export SECRETS_ENCRYPTION_KEY="dev-32-bytes-secrets-key-apfninj"
    sed -i "s/SECRETS_ENCRYPTION_KEY=.*/SECRETS_ENCRYPTION_KEY=$SECRETS_ENCRYPTION_KEY/" "$CONFIG_FILE"
fi


if [ -z "$CLOUDSQL_PASSWORD" ]; then
    echo "================================================================="
    echo "🔒 Security Setup: CLOUDSQL_PASSWORD is required for database access."
    echo "Please enter your Cloud SQL Password (it will be saved securely in config.env):"
    read -s -p "Password: " USER_DB_PASS
    echo ""
    if [ -n "$USER_DB_PASS" ]; then
        sed -i "s/CLOUDSQL_PASSWORD=.*/CLOUDSQL_PASSWORD=$USER_DB_PASS/" "$CONFIG_FILE"
        export CLOUDSQL_PASSWORD="$USER_DB_PASS"
        echo "✅ Cloud SQL Password saved to config.env."
    else
        echo "⚠️ No Password entered. Database connections may fail to authenticate."
    fi
    echo "================================================================="
fi

# Ensure CLOUDSQL_RUNNER_DATABASE exists in config.env for runner isolation
if ! grep -q "^CLOUDSQL_RUNNER_DATABASE=" "$CONFIG_FILE"; then
    echo "CLOUDSQL_RUNNER_DATABASE=apf-local-runner-session-db" >> "$CONFIG_FILE"
fi

# Ensure SIMULATED_USER_ID entry exists in config.env
if ! grep -q "^SIMULATED_USER_ID=" "$CONFIG_FILE"; then
    echo "SIMULATED_USER_ID=" >> "$CONFIG_FILE"
fi

if [ -z "$SIMULATED_USER_ID" ]; then
    echo "================================================================="
    echo "👤 Setup: SIMULATED_USER_ID is required for local user session simulation."
    
    # Attempt to get default user from gcloud
    DEFAULT_USER=""
    if command -v gcloud >/dev/null 2>&1; then
        DEFAULT_USER=$(gcloud info --format="value(config.account)" 2>/dev/null || true)
    fi
    
    if [ -n "$DEFAULT_USER" ]; then
        read -p "Enter SIMULATED_USER_ID [$DEFAULT_USER]: " USER_SIMULATED_ID
        USER_SIMULATED_ID=${USER_SIMULATED_ID:-$DEFAULT_USER}
    else
        echo "Suggestion: Use the email id from your chrome profile that you will typically use to test the app in browser"
        read -p "Enter SIMULATED_USER_ID: " USER_SIMULATED_ID
    fi
    echo ""
    if [ -n "$USER_SIMULATED_ID" ]; then
        sed -i "s/SIMULATED_USER_ID=.*/SIMULATED_USER_ID=$USER_SIMULATED_ID/" "$CONFIG_FILE"
        export SIMULATED_USER_ID="$USER_SIMULATED_ID"
        echo "✅ SIMULATED_USER_ID saved to config.env."
    else
        echo "⚠️ No SIMULATED_USER_ID entered. User simulation may not work correctly."
    fi
    echo "================================================================="
fi

if [ -z "$SUPER_ADMIN_EMAILS" ]; then
    echo "================================================================="
    echo "👤 Setup: SUPER_ADMIN_EMAILS is required for admin access."
    
    # Attempt to get default user from gcloud
    DEFAULT_ADMIN=""
    if command -v gcloud >/dev/null 2>&1; then
        DEFAULT_ADMIN=$(gcloud info --format="value(config.account)" 2>/dev/null || true)
    fi
    
    if [ -n "$DEFAULT_ADMIN" ]; then
        read -p "Enter SUPER_ADMIN_EMAILS [$DEFAULT_ADMIN]: " USER_SUPER_ADMINS
        USER_SUPER_ADMINS=${USER_SUPER_ADMINS:-$DEFAULT_ADMIN}
    else
        read -p "Enter SUPER_ADMIN_EMAILS (comma-separated): " USER_SUPER_ADMINS
    fi
    echo ""
    if [ -n "$USER_SUPER_ADMINS" ]; then
        sed -i "s/SUPER_ADMIN_EMAILS=.*/SUPER_ADMIN_EMAILS=$USER_SUPER_ADMINS/" "$CONFIG_FILE"
        export SUPER_ADMIN_EMAILS="$USER_SUPER_ADMINS"
        echo "✅ SUPER_ADMIN_EMAILS saved to config.env."
    else
        echo "⚠️ No SUPER_ADMIN_EMAILS entered."
    fi
    echo "================================================================="
fi

# Ensure Epic 11 IAM variables exist in config.env
if ! grep -q "^ENVIRONMENT=" "$CONFIG_FILE"; then
    echo "ENVIRONMENT=local" >> "$CONFIG_FILE"
fi
if ! grep -q "^BYPASS_IAP_VALIDATION=" "$CONFIG_FILE"; then
    echo "BYPASS_IAP_VALIDATION=true" >> "$CONFIG_FILE"
fi
if ! grep -q "^JWT_SECRET=" "$CONFIG_FILE"; then
    echo "JWT_SECRET=local-development-secret-key-32-chars-long" >> "$CONFIG_FILE"
fi
if ! grep -q "^IAP_AUDIENCE=" "$CONFIG_FILE"; then
    echo "IAP_AUDIENCE=" >> "$CONFIG_FILE"
fi
if ! grep -q "^SUPER_ADMIN_EMAILS=" "$CONFIG_FILE"; then
    echo "SUPER_ADMIN_EMAILS=" >> "$CONFIG_FILE"
fi
if ! grep -q "^FIREBASE_AUTH_EMULATOR_HOST=" "$CONFIG_FILE"; then
    echo "FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:8085" >> "$CONFIG_FILE"
fi
if ! grep -q "^NEXT_PUBLIC_ENABLE_MOCK_AUTH=" "$CONFIG_FILE"; then
    echo "NEXT_PUBLIC_ENABLE_MOCK_AUTH=false" >> "$CONFIG_FILE"
fi
if ! grep -q "^NEXT_PUBLIC_FIREBASE_API_KEY=" "$CONFIG_FILE"; then
    echo "NEXT_PUBLIC_FIREBASE_API_KEY=local-mock-api-key-for-emulator-boot" >> "$CONFIG_FILE"
fi
if ! grep -q "^NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN=" "$CONFIG_FILE"; then
    echo "NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN=your-firebase-project.firebaseapp.com" >> "$CONFIG_FILE"
fi
if ! grep -q "^NEXT_PUBLIC_FIREBASE_PROJECT_ID=" "$CONFIG_FILE"; then
    echo "NEXT_PUBLIC_FIREBASE_PROJECT_ID=local-testing-mock-project" >> "$CONFIG_FILE"
fi
if ! grep -q "^NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST=" "$CONFIG_FILE"; then
    echo "NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:8085" >> "$CONFIG_FILE"
fi

# Ensure Inter-Service RSA Encryption Keys exist in config.env
if [ -z "$SECRET_ENCRYPTION_PRIVATE_KEY" ] || [ -z "$SECRET_ENCRYPTION_PUBLIC_KEY" ]; then
    echo "================================================================="
    echo "🔐 Generating unique RSA 2048 key pair for inter-service secret encryption..."
    TMP_KEY_DIR=$(mktemp -d)
    openssl genpkey -algorithm RSA -out "$TMP_KEY_DIR/private.pem" -pkeyopt rsa_keygen_bits:2048 2>/dev/null || openssl genrsa -out "$TMP_KEY_DIR/private.pem" 2048 2>/dev/null
    openssl rsa -in "$TMP_KEY_DIR/private.pem" -pubout -out "$TMP_KEY_DIR/public.pem" 2>/dev/null

    GEN_PRIV_KEY=$(awk '{printf "%s\\n", $0}' "$TMP_KEY_DIR/private.pem")
    GEN_PUB_KEY=$(awk '{printf "%s\\n", $0}' "$TMP_KEY_DIR/public.pem")
    rm -rf "$TMP_KEY_DIR"

    sed -i '/^SECRET_ENCRYPTION_PRIVATE_KEY=/d' "$CONFIG_FILE"
    sed -i '/^SECRET_ENCRYPTION_PUBLIC_KEY=/d' "$CONFIG_FILE"
    sed -i '/^PROXY_PRIVATE_KEY=/d' "$CONFIG_FILE"
    sed -i '/^PROXY_PUBLIC_KEY=/d' "$CONFIG_FILE"
    sed -i '/^ALLOW_DEV_FALLBACK_KEY=/d' "$CONFIG_FILE"

    echo "" >> "$CONFIG_FILE"
    echo "# Inter-Service RSA Encryption Keys (Generated automatically for local development)" >> "$CONFIG_FILE"
    echo "SECRET_ENCRYPTION_PRIVATE_KEY=\"$GEN_PRIV_KEY\"" >> "$CONFIG_FILE"
    echo "SECRET_ENCRYPTION_PUBLIC_KEY=\"$GEN_PUB_KEY\"" >> "$CONFIG_FILE"

    export SECRET_ENCRYPTION_PRIVATE_KEY="$GEN_PRIV_KEY"
    export SECRET_ENCRYPTION_PUBLIC_KEY="$GEN_PUB_KEY"
    echo "✅ Unique RSA key pair generated and saved to config.env."
    echo "================================================================="
fi

# Re-source config.env to load newly appended variables
source "$CONFIG_FILE"


echo "Checking for Firebase CLI (firebase-tools)..."
if command -v firebase >/dev/null 2>&1; then
    echo "✅ Firebase CLI is already installed at $(command -v firebase)."
else
    echo "⚠️ Firebase CLI not found in PATH. Installing firebase-tools via npm..."
    if command -v npm >/dev/null 2>&1; then
        npm install -g firebase-tools || sudo npm install -g firebase-tools || echo "❌ Failed to install firebase-tools globally. Please run 'npm install -g firebase-tools' manually."
        echo "✅ Firebase CLI installed successfully."
    else
        echo "❌ npm not found. Cannot install firebase-tools. Please install Node.js/npm first."
    fi
fi

echo "Generating ai/.env..."
mkdir -p "$ROOT_DIR/ai"
cat <<EOF > "$ROOT_DIR/ai/.env"
# Generated automatically by scripts/setup_env.sh from root config.env
GEMINI_API_KEY=$GEMINI_API_KEY
SERVICE_NAME=$SERVICE_NAME
AGENT_LOADER_NAME=$AGENT_LOADER_NAME
ARTIFACT_SERVICE_TYPE=$ARTIFACT_SERVICE_TYPE
GCS_ARTIFACT_BUCKET=$GCS_ARTIFACT_BUCKET
GITHUB_ACCESS_TOKEN=$GITHUB_ACCESS_TOKEN
SECRET_ENCRYPTION_PRIVATE_KEY="$SECRET_ENCRYPTION_PRIVATE_KEY"
WORKSPACE_MOUNT_PATH=$WORKSPACE_MOUNT_PATH
LOCAL_ARTIFACT_PATH=$LOCAL_ARTIFACT_PATH
SESSION_SERVICE_TYPE=$SESSION_SERVICE_TYPE
CLOUDSQL_CONNECTION_NAME=$CLOUDSQL_CONNECTION_NAME
CLOUDSQL_USER=$CLOUDSQL_USER
CLOUDSQL_PASSWORD=$CLOUDSQL_PASSWORD
CLOUDSQL_DATABASE=$CLOUDSQL_DATABASE
DEFAULT_LLM_MODEL=$DEFAULT_LLM_MODEL
AGENT_BASE_URL=$AGENT_BASE_URL
PORT=$PORT
SIMULATED_USER_ID=$SIMULATED_USER_ID
BMAD_ROOT_PATH=${BMAD_ROOT_PATH:-$HOME/.apf/config/bmad}
EOF
echo "Successfully populated ai/.env"

echo "Generating api/.env..."
mkdir -p "$ROOT_DIR/api"
cat <<EOF > "$ROOT_DIR/api/.env"
# Generated automatically by scripts/setup_env.sh from root config.env
GOOGLE_CLOUD_PROJECT=$GOOGLE_CLOUD_PROJECT
GOOGLE_CLOUD_LOCATION=$GOOGLE_CLOUD_LOCATION
GEMINI_API_KEY=$GEMINI_API_KEY
FIRESTORE_EMULATOR_HOST=$FIRESTORE_EMULATOR_HOST
FIREBASE_AUTH_EMULATOR_HOST=$FIREBASE_AUTH_EMULATOR_HOST
FIRESTORE_DB_NAME=$FIRESTORE_DB_NAME
WORKSPACE_MOUNT_PATH=$WORKSPACE_MOUNT_PATH
LOCAL_ARTIFACT_PATH=$LOCAL_ARTIFACT_PATH
GCS_ARTIFACT_BUCKET=$GCS_ARTIFACT_BUCKET
ARTIFACT_SERVICE_TYPE=$ARTIFACT_SERVICE_TYPE
AGENT_BASE_URL=$AGENT_BASE_URL
PORT=$API_PORT
PERF_TEST_MODE=$PERF_TEST_MODE
LOG_LEVEL=$LOG_LEVEL
SIMULATED_USER_ID=$SIMULATED_USER_ID
GITHUB_ACCESS_TOKEN=$GITHUB_ACCESS_TOKEN
DEV_REPO_ACCESS_TOKEN=$DEV_REPO_ACCESS_TOKEN
SECRETS_ENCRYPTION_KEY=$SECRETS_ENCRYPTION_KEY
SECRET_ENCRYPTION_PUBLIC_KEY="$SECRET_ENCRYPTION_PUBLIC_KEY"
ENVIRONMENT=$ENVIRONMENT
BYPASS_IAP_VALIDATION=$BYPASS_IAP_VALIDATION
JWT_SECRET=$JWT_SECRET
IAP_AUDIENCE=$IAP_AUDIENCE
SUPER_ADMIN_EMAILS=$SUPER_ADMIN_EMAILS
CLOUDSQL_CONNECTION_NAME=$CLOUDSQL_CONNECTION_NAME
CLOUDSQL_USER=$CLOUDSQL_USER
CLOUDSQL_PASSWORD=$CLOUDSQL_PASSWORD
CLOUDSQL_DATABASE=$CLOUDSQL_DATABASE
BMAD_ROOT_PATH=${BMAD_ROOT_PATH:-$HOME/.apf/config/bmad}
EOF
echo "Successfully populated api/.env"


echo "Generating ui/.env.local..."
mkdir -p "$ROOT_DIR/ui"
cat <<EOF > "$ROOT_DIR/ui/.env.local"
# Generated automatically by scripts/setup_env.sh from root config.env
API_BACKEND=http://localhost:$API_PORT
PORT=8082
NEXT_PUBLIC_ENABLE_MOCK_AUTH=$NEXT_PUBLIC_ENABLE_MOCK_AUTH
NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST=$NEXT_PUBLIC_FIREBASE_AUTH_EMULATOR_HOST
NEXT_PUBLIC_FIREBASE_API_KEY=$NEXT_PUBLIC_FIREBASE_API_KEY
NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN=$NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN
NEXT_PUBLIC_FIREBASE_PROJECT_ID=$NEXT_PUBLIC_FIREBASE_PROJECT_ID
EOF
echo "Successfully populated ui/.env.local"

if [ -d "$ROOT_DIR/runners/adk" ]; then
    echo "Generating runners/adk/.env..."
    cat <<EOF > "$ROOT_DIR/runners/adk/.env"
# Generated automatically by scripts/setup_env.sh from root config.env
GOOGLE_CLOUD_PROJECT=$GOOGLE_CLOUD_PROJECT
GOOGLE_CLOUD_LOCATION=$GOOGLE_CLOUD_LOCATION
GEMINI_API_KEY=$GEMINI_API_KEY
WORKSPACE_MOUNT_PATH=$WORKSPACE_MOUNT_PATH
DEFAULT_LLM_MODEL=$DEFAULT_LLM_MODEL
LOG_LEVEL=$LOG_LEVEL
CLOUDSQL_CONNECTION_NAME=$CLOUDSQL_CONNECTION_NAME
CLOUDSQL_USER=$CLOUDSQL_USER
CLOUDSQL_PASSWORD=$CLOUDSQL_PASSWORD
CLOUDSQL_DATABASE=$CLOUDSQL_DATABASE
SESSION_SERVICE_TYPE=$SESSION_SERVICE_TYPE
ARTIFACT_BUCKET_NAME=$GCS_ARTIFACT_BUCKET
BMAD_ROOT_PATH=${BMAD_ROOT_PATH:-$HOME/.apf/config/bmad}
EOF
    echo "Successfully populated runners/adk/.env"
fi

echo "Checking for Node dependencies in ui/..."
if [ ! -d "$ROOT_DIR/ui/node_modules" ]; then
    echo "⚠️ node_modules not found in ui/. Running npm install..."
    cd "$ROOT_DIR/ui" && npm install --legacy-peer-deps
    cd "$ROOT_DIR"
    echo "✅ Node dependencies installed successfully."
fi

# Create a backup copy of master config.env in ~/.apf/config.env.bak
mkdir -p "$HOME/.apf"
cp "$CONFIG_FILE" "$HOME/.apf/config.env.bak"
echo "✅ Backup copy of config.env saved to $HOME/.apf/config.env.bak"

echo "================================================================="
echo "Environment setup and synchronization completed successfully!"
echo "================================================================="
