#!/bin/bash
# Setup rtuccli alias for Linux/macOS

SHELL_RC=""
ALIAS_LINE='alias rtuccli="docker-compose exec -T admin-service sh -c"'

# Detect shell
if [ -n "$BASH_VERSION" ]; then
    SHELL_RC="$HOME/.bashrc"
elif [ -n "$ZSH_VERSION" ]; then
    SHELL_RC="$HOME/.zshrc"
else
    echo "Unknown shell. Please add the alias manually to your shell config:"
    echo "$ALIAS_LINE"
    exit 1
fi

# Check if alias already exists
if grep -q "alias rtuccli=" "$SHELL_RC" 2>/dev/null; then
    echo "✓ Alias 'rtuccli' already exists in $SHELL_RC"
else
    echo "Adding rtuccli alias to $SHELL_RC..."
    echo "" >> "$SHELL_RC"
    echo "# rtuccli - chat-smpl CLI tool" >> "$SHELL_RC"
    echo "$ALIAS_LINE" >> "$SHELL_RC"
    echo "✓ Alias added to $SHELL_RC"
fi

# Try to reload shell config
if [ -n "$BASH_VERSION" ]; then
    source "$SHELL_RC" 2>/dev/null || true
elif [ -n "$ZSH_VERSION" ]; then
    source "$SHELL_RC" 2>/dev/null || true
fi

echo ""
echo "✓ Setup complete!"
echo ""
echo "To use the alias immediately, run:"
echo "  source $SHELL_RC"
echo ""
echo "Or open a new terminal window."
echo ""
echo "Usage examples:"
echo "  rtuccli 'service list'"
echo "  rtuccli 'conf list --status active'"
echo ""
