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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Configuration file resolution
CONFIG_FILE="${DEVLOOP_CONFIG:-$REPO_ROOT/devloop.config}"
if [ ! -f "$CONFIG_FILE" ] && [ -f "$PWD/devloop.config" ]; then
    CONFIG_FILE="$PWD/devloop.config"
fi

# Requirement 3: Fail with appropriate message if devloop.config is not present
if [ ! -f "$CONFIG_FILE" ]; then
    echo "=================================================================" >&2
    echo "❌ Error: Configuration file 'devloop.config' was not found!" >&2
    echo "Expected location: $CONFIG_FILE" >&2
    echo "" >&2
    echo "Please create a 'devloop.config' file with the required configuration" >&2
    echo "before invoking 'mise run devloop-cli'." >&2
    echo "=================================================================" >&2
    exit 1
fi

# Requirement 1 & 4: Read and show all configuration values from devloop.config
echo "================================================================="
echo "⚙️  Loaded DevLoop Configuration from: $CONFIG_FILE"
echo "================================================================="

# Parse config using Python (supports YAML, JSON, and KEY=VALUE dotenv format)
CONFIG_JSON=$(python3 - <<EOF
import sys, os, json

config_path = "$CONFIG_FILE"

try:
    with open(config_path, "r", encoding="utf-8") as f:
        content = f.read()
except Exception as e:
    sys.stderr.write(f"❌ Error reading config file {config_path}: {e}\n")
    sys.exit(1)

data = {}

# Try YAML/JSON parsing first
try:
    import yaml
    loaded = yaml.safe_load(content)
    if isinstance(loaded, dict):
        data = loaded
except Exception:
    pass

# Fallback: line-by-line key=value or key: value parser
if not data:
    for line in content.splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if "=" in line:
            k, v = line.split("=", 1)
            data[k.strip().strip("\"'")] = v.strip().strip("\"'")
        elif ":" in line:
            k, v = line.split(":", 1)
            data[k.strip().strip("\"'")] = v.strip().strip("\"'")

# Normalize case for standard keys
def get_val(keys, default=""):
    for k in keys:
        for existing_key in data:
            if existing_key.lower().replace("_", "").replace("-", "") == k.lower().replace("_", "").replace("-", ""):
                val = data[existing_key]
                if isinstance(val, bool):
                    return "true" if val else "false"
                return str(val)
    return default

repo_root = "$REPO_ROOT"
default_python = os.path.join(repo_root, "runners/adk/.venv/bin/python")
if not os.path.exists(default_python):
    default_python = sys.executable

# Display table of configurations to stderr
if data:
    for k, v in data.items():
        print(f"  {str(k):<32}: {str(v)}", file=sys.stderr)
else:
    print("  (Empty configuration file)", file=sys.stderr)

resolved = {
    "DEBUG": get_val(["debug"], "true"),
    "ENV_FILE_PATH": get_val(["env_file_path", "envfilepath", "envfile"], os.path.join(repo_root, "runners/adk/.env")),
    "TARGET_FOLDER": get_val(["target_folder", "targetfolder"], os.path.expanduser("~/.apf/org-high-council/simulated-super-admin-uid/agentic-foundry")),
    "BMAD_SRC_ROOT": get_val(["bmad_src_root", "bmadsrcroot", "bmad_root"], os.path.expanduser("~/.apf/config/bmad")),
    "SESSION_ID": get_val(["session_id", "sessionid"], ""),
    "APP_NAME": get_val(["app_name", "appname"], "dev_loop_workflow"),
    "USER_ID": get_val(["user_id", "userid"], "cli_user"),
    "PYTHON_BIN": get_val(["python_bin", "pythonbin", "python"], default_python),
    "RUN_AGENT_CLI": get_val(["run_agent_cli", "runagentcli"], os.path.join(repo_root, "runners/adk/run_agent_cli.py")),
    "RAW_CONFIG": {str(k): str(v) for k, v in data.items()}
}

print(json.dumps(resolved))
EOF
)

if [ $? -ne 0 ]; then
    echo "❌ Error: Failed to parse configuration file $CONFIG_FILE" >&2
    exit 1
fi

echo "================================================================="

# Requirement 4: Prompt user for y/n confirmation
CONFIRM=""
if [ "$ASSUME_YES" = "true" ] || [ "$NON_INTERACTIVE" = "true" ] || [ "$1" = "-y" ] || [ "$1" = "--yes" ]; then
    CONFIRM="y"
    echo "Auto-confirming configuration (non-interactive mode): yes"
else
    read -r -p "Proceed with these configuration settings? (y/n): " CONFIRM
fi

case "$CONFIRM" in
    [yY]|[yY][eE][sS])
        echo "✅ Configuration confirmed. Starting DevLoop CLI..."
        ;;
    *)
        echo "🛑 DevLoop execution cancelled by user."
        exit 0
        ;;
esac

# Extract resolved parameters using CONFIG_JSON
export CONFIG_JSON
RESOLVED_DEBUG=$(python3 -c "import os, json; print(json.loads(os.environ['CONFIG_JSON']).get('DEBUG', 'true'))")
RESOLVED_ENV_FILE_PATH=$(python3 -c "import os, json; print(json.loads(os.environ['CONFIG_JSON']).get('ENV_FILE_PATH', ''))")
RESOLVED_TARGET_FOLDER=$(python3 -c "import os, json; print(json.loads(os.environ['CONFIG_JSON']).get('TARGET_FOLDER', ''))")
RESOLVED_BMAD_SRC_ROOT=$(python3 -c "import os, json; print(json.loads(os.environ['CONFIG_JSON']).get('BMAD_SRC_ROOT', ''))")
RESOLVED_SESSION_ID=$(python3 -c "import os, json; print(json.loads(os.environ['CONFIG_JSON']).get('SESSION_ID', ''))")
RESOLVED_APP_NAME=$(python3 -c "import os, json; print(json.loads(os.environ['CONFIG_JSON']).get('APP_NAME', 'dev_loop_workflow'))")
RESOLVED_USER_ID=$(python3 -c "import os, json; print(json.loads(os.environ['CONFIG_JSON']).get('USER_ID', 'cli_user'))")
RESOLVED_PYTHON_BIN=$(python3 -c "import os, json; print(json.loads(os.environ['CONFIG_JSON']).get('PYTHON_BIN', ''))")
RESOLVED_RUN_AGENT_CLI=$(python3 -c "import os, json; print(json.loads(os.environ['CONFIG_JSON']).get('RUN_AGENT_CLI', ''))")

if [ -z "$RESOLVED_SESSION_ID" ]; then
    RESOLVED_SESSION_ID="devloop_cli_$(date +%s)"
fi

if [ ! -f "$RESOLVED_PYTHON_BIN" ]; then
    RESOLVED_PYTHON_BIN="python3"
fi

echo "🚀 Executing DevLoop CLI session: $RESOLVED_SESSION_ID (App: $RESOLVED_APP_NAME, Target: $RESOLVED_TARGET_FOLDER)"

DEBUG="$RESOLVED_DEBUG" \
ENV_FILE_PATH="$RESOLVED_ENV_FILE_PATH" \
TARGET_FOLDER="$RESOLVED_TARGET_FOLDER" \
BMAD_SRC_ROOT="$RESOLVED_BMAD_SRC_ROOT" \
"$RESOLVED_PYTHON_BIN" "$RESOLVED_RUN_AGENT_CLI" \
"{\"appName\":\"$RESOLVED_APP_NAME\",\"userId\":\"$RESOLVED_USER_ID\",\"sessionId\":\"$RESOLVED_SESSION_ID\"}" "$RESOLVED_TARGET_FOLDER"

