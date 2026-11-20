#!/usr/bin/env bash
set -euo pipefail

if ! command -v jq >/dev/null 2>&1; then
  echo "block-sensitive-paths: jq is required to evaluate tool guardrails" >&2
  exit 2
fi

INPUT="$(cat)"

if ! TOOL_NAME="$(printf '%s' "$INPUT" | jq -r '.tool_name // empty')"; then
  echo "block-sensitive-paths: failed to parse tool hook payload" >&2
  exit 2
fi

check_value() {
  local val="$1"
  [[ -z "$val" ]] && return 0

  # Block .env and .env.* files
  if [[ "$val" =~ (^|/|[[:space:]\"\'\=])\.env([._-]|$|[[:space:]/\"\'\;\|\&\>]) ]]; then
    echo "Blocked access to sensitive .env path: $val" >&2
    exit 2
  fi

  # Block *.key files
  if [[ "$val" =~ \.key($|[[:space:]\"\'\;\|\&\>]) ]]; then
    echo "Blocked access to private key file (*.key): $val" >&2
    exit 2
  fi

  # Block decrypted secret artifacts under secrets/*.dec.*
  if [[ "$val" =~ (^|/|[[:space:]\"\'\=])secrets/[^/[:space:]\"\']*\.dec\. ]]; then
    echo "Blocked access to decrypted secret artifact (secrets/*.dec.*): $val" >&2
    exit 2
  fi

  # Block SSH directory (~/.ssh, $HOME/.ssh, /home/*/.ssh, /root/.ssh, .ssh/)
  if [[ "$val" =~ (^|[[:space:]\"\'\=])(~|\$HOME|\$\{HOME\}|/home/[^/[:space:]\"\']+|/root)?/\.ssh(/|$|[[:space:]\"\'\;\|\&\>]) ]] ||
     [[ "$val" =~ (^|[[:space:]\"\'\=])\.ssh(/|$|[[:space:]\"\'\;\|\&\>]) ]]; then
    echo "Blocked access to SSH directory (~/.ssh/**): $val" >&2
    exit 2
  fi
}

case "$TOOL_NAME" in
  Read|Edit|Write|Grep)
    while IFS= read -r val; do
      check_value "$val"
    done < <(printf '%s' "$INPUT" | jq -r '(.tool_input.file_path // empty), (.tool_input.path // empty), (.tool_input.glob // empty)')
    ;;
  Glob)
    while IFS= read -r val; do
      check_value "$val"
    done < <(printf '%s' "$INPUT" | jq -r '(.tool_input.file_path // empty), (.tool_input.path // empty), (.tool_input.glob // empty), (.tool_input.pattern // empty)')
    ;;
  Bash)
    CMD="$(printf '%s' "$INPUT" | jq -r '.tool_input.command // empty')"
    check_value "$CMD"
    ;;
esac

exit 0
