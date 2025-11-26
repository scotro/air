#!/bin/bash
#
# AI Agent Workflow - Installation Script
#
# This script sets up the agent-helpers for use in any terminal.
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "AI Agent Workflow - Installation"
echo "================================="
echo ""

# Determine the script's directory (where ai-workflow is cloned)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HELPERS_PATH="$SCRIPT_DIR/agent-helpers.sh"

if [[ ! -f "$HELPERS_PATH" ]]; then
    echo -e "${RED}Error: agent-helpers.sh not found in $SCRIPT_DIR${NC}"
    echo "Make sure you're running this from the ai-workflow directory."
    exit 1
fi

# Detect user's login shell (not the shell running this script)
SHELL_NAME="$(basename "$SHELL")"

case "$SHELL_NAME" in
    zsh)  SHELL_RC="$HOME/.zshrc" ;;
    bash) SHELL_RC="$HOME/.bashrc" ;;
    *)
        echo -e "${YELLOW}Unsupported shell: $SHELL_NAME${NC}"
        echo "Manually add this line to your shell config:"
        echo "  source \"$HELPERS_PATH\""
        exit 0
        ;;
esac

echo "Detected shell: $SHELL_NAME"
echo "Config file: $SHELL_RC"
echo ""

# Check if already installed
if grep -q "agent-helpers.sh" "$SHELL_RC" 2>/dev/null; then
    echo -e "${YELLOW}Already installed!${NC}"
    echo "agent-helpers.sh is already sourced in $SHELL_RC"
    echo ""
    echo "To update, pull the latest changes:"
    echo "  cd $SCRIPT_DIR && git pull"
    echo ""
    echo "To reinstall, remove the line from $SHELL_RC and run again."
    exit 0
fi

# Add source line to shell config
echo "Adding to $SHELL_RC..."
echo "" >> "$SHELL_RC"
echo "# AI Agent Workflow helpers" >> "$SHELL_RC"
echo "source \"$HELPERS_PATH\"" >> "$SHELL_RC"

echo -e "${GREEN}Installation complete!${NC}"
echo ""
echo "To start using the helpers, either:"
echo "  1. Open a new terminal, or"
echo "  2. Run: source $SHELL_RC"
echo ""
echo "Quick start:"
echo "  cd ~/your-project"
echo "  agent-init          # Initialize project for workflow"
echo "  agent-help          # See all commands"
echo ""
echo "Documentation:"
echo "  $SCRIPT_DIR/README.md"
echo "  $SCRIPT_DIR/CHEATSHEET.md"
