#!/usr/bin/env bash
# Interactive setup script for Telegram messenger integration with APS profiles
#
# Guides users through:
# 1. Creating/selecting a profile
# 2. Creating/selecting a Telegram messenger
# 3. Configuring channel-to-action mappings
# 4. Setting up and starting the messenger
#
# Usage:
#   ./scripts/setup-telegram.sh

set -euo pipefail

# Colors for terminal output
if [ -t 1 ]; then
  GREEN='\033[0;32m'
  RED='\033[0;31m'
  CYAN='\033[0;36m'
  YELLOW='\033[0;33m'
  BOLD='\033[1m'
  RESET='\033[0m'
else
  GREEN='' RED='' CYAN='' YELLOW='' BOLD='' RESET=''
fi

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

# Global state
PROFILE_ID=""
MESSENGER_NAME=""
TELEGRAM_TOKEN=""
CHANNELS=()
DEFAULT_ACTION=""

# Helper functions
print_header() {
  echo -e "\n${BOLD}${CYAN}==== $1 ====${RESET}\n"
}

print_step() {
  echo -e "${BOLD}$1${RESET}"
}

print_info() {
  echo -e "${CYAN}ℹ$RESET  $1"
}

print_success() {
  echo -e "${GREEN}✓${RESET}  $1"
}

print_error() {
  echo -e "${RED}✗${RESET}  $1" >&2
}

print_warning() {
  echo -e "${YELLOW}⚠${RESET}  $1"
}

prompt() {
  local message="$1"
  local default="${2:-}"
  local result

  if [ -n "$default" ]; then
    echo -n -e "${BOLD}${message}${RESET} [${CYAN}${default}${RESET}]: "
  else
    echo -n -e "${BOLD}${message}${RESET}: "
  fi
  read -r result || result=""

  if [ -z "$result" ] && [ -n "$default" ]; then
    echo "$default"
  else
    echo "$result"
  fi
}

prompt_yn() {
  local message="$1"
  local result

  while true; do
    echo -n -e "${BOLD}${message}${RESET} [${CYAN}y/n${RESET}]: "
    read -r result || result=""
    case "$result" in
      [Yy]*) return 0 ;;
      [Nn]*) return 1 ;;
      *) echo "Please answer y or n" ;;
    esac
  done
}

# Check prerequisites
check_prerequisites() {
  print_header "Checking Prerequisites"

  if ! command -v aps &> /dev/null; then
    print_error "aps CLI not found in PATH"
    echo "Please install APS and ensure it's in your PATH"
    exit 1
  fi
  print_success "aps CLI found"

  if ! aps profile list &> /dev/null; then
    print_warning "No profiles found yet"
  else
    print_success "APS profiles accessible"
  fi
}

# Step 1: Select or create profile
setup_profile() {
  print_header "Step 1: Profile Selection"

  local profiles
  profiles=$(aps profile list 2>/dev/null || echo "")

  if [ -n "$profiles" ]; then
    print_info "Existing profiles:"
    echo "$profiles" | while read -r profile; do
      echo "  • $profile"
    done
    echo ""

    if prompt_yn "Use an existing profile?"; then
      PROFILE_ID=$(prompt "Profile ID")
    else
      PROFILE_ID=$(prompt "Create new profile with ID")
      print_info "Creating profile: $PROFILE_ID"
      if aps profile create "$PROFILE_ID"; then
        print_success "Profile created"
      else
        print_error "Failed to create profile"
        exit 1
      fi
    fi
  else
    PROFILE_ID=$(prompt "Create new profile with ID")
    print_info "Creating profile: $PROFILE_ID"
    if aps profile create "$PROFILE_ID"; then
      print_success "Profile created"
    else
      print_error "Failed to create profile"
      exit 1
    fi
  fi

  print_success "Using profile: ${CYAN}${PROFILE_ID}${RESET}"
}

# Step 2: Select or create Telegram messenger
setup_messenger() {
  print_header "Step 2: Telegram Messenger Setup"

  MESSENGER_NAME=$(prompt "Telegram messenger name" "my-telegram")
  print_success "Messenger name: ${CYAN}${MESSENGER_NAME}${RESET}"

  # Check if messenger already exists
  if [ -f "$HOME/.aps/messengers/${MESSENGER_NAME}/config.yaml" ]; then
    print_warning "Messenger '${MESSENGER_NAME}' already exists"
    if prompt_yn "Configure existing messenger?"; then
      print_info "Reusing existing messenger"
      return
    fi
  fi

  print_info "Creating new Telegram messenger..."
  echo ""

  local template
  if prompt_yn "Use subprocess mode (always listening for messages)?"; then
    template="subprocess"
    print_info "Messenger will run continuously and listen for Telegram updates"
  else
    template="webhook"
    print_info "Messenger will receive messages via webhook triggers"
  fi
  echo ""

  local language
  language=$(prompt "Preferred language for messenger implementation" "python")

  print_info "Creating messenger: $MESSENGER_NAME (${language}, ${template} mode)"
  if ! aps messengers create "$MESSENGER_NAME" \
    --template="$template" \
    --language="$language" 2>/dev/null; then
    print_error "Failed to create messenger"
    exit 1
  fi
  print_success "Messenger created"

  # Collect Telegram token
  echo ""
  print_info "You need a Telegram bot token. Get one from BotFather:"
  echo "  1. Open Telegram and search for @BotFather"
  echo "  2. Send /newbot and follow the steps"
  echo "  3. Copy the bot token (looks like: 123456789:ABCDefGHijKLmnoPQRstUVwxYZ)"
  echo ""
  TELEGRAM_TOKEN=$(prompt "Telegram bot token")

  if [ -z "$TELEGRAM_TOKEN" ]; then
    print_error "Token cannot be empty"
    exit 1
  fi
  print_success "Token stored"
}

# Step 3: Configure channel mappings
setup_channel_mappings() {
  print_header "Step 3: Configure Channel Mappings"

  print_info "Map Telegram channels/groups to profile actions"
  echo ""
  print_info "Channel ID format for Telegram:"
  echo "  • Direct messages: positive number (e.g., 123456789)"
  echo "  • Groups/supergroups: negative number (e.g., -1001234567890)"
  echo ""

  while true; do
    local channel_id
    channel_id=$(prompt "Telegram channel ID (or 'done' to finish)")

    if [ "$channel_id" = "done" ] || [ "$channel_id" = "Done" ]; then
      break
    fi

    if [ -z "$channel_id" ]; then
      print_warning "Channel ID cannot be empty"
      continue
    fi

    local action
    action=$(prompt "Action to execute (e.g., handle-telegram, process-command)")

    if [ -z "$action" ]; then
      print_warning "Action cannot be empty"
      continue
    fi

    local mapping="${PROFILE_ID}=${action}"
    CHANNELS+=("${channel_id}:${mapping}")
    print_success "Mapped channel ${CYAN}${channel_id}${RESET} → ${CYAN}${action}${RESET}"
    echo ""
  done

  if [ ${#CHANNELS[@]} -eq 0 ]; then
    print_warning "No channel mappings configured"
    if prompt_yn "Set a default action instead?"; then
      DEFAULT_ACTION=$(prompt "Default action")
      print_success "Default action set to: ${CYAN}${DEFAULT_ACTION}${RESET}"
    else
      print_warning "No actions mapped; messenger will ignore all messages"
    fi
  else
    print_success "Configured ${CYAN}${#CHANNELS[@]}${RESET} channel mapping(s)"

    echo ""
    if prompt_yn "Set a default action for unmapped channels?"; then
      DEFAULT_ACTION=$(prompt "Default action")
      print_success "Default action: ${CYAN}${DEFAULT_ACTION}${RESET}"
    fi
  fi
}

# Step 4: Apply configuration
apply_configuration() {
  print_header "Step 4: Applying Configuration"

  print_step "1. Linking messenger to profile..."

  # Build mappings arg
  local mappings_args=()
  for channel_mapping in "${CHANNELS[@]}"; do
    local channel_id
    local mapping
    IFS=':' read -r channel_id mapping <<< "$channel_mapping"
    mappings_args+=("--channel" "${channel_id}=${mapping}")
  done

  if [ -n "$DEFAULT_ACTION" ]; then
    mappings_args+=("--default-action" "$DEFAULT_ACTION")
  fi

  if ! aps profile link-messenger "$PROFILE_ID" "$MESSENGER_NAME" "${mappings_args[@]}" 2>/dev/null; then
    print_error "Failed to link messenger to profile"
    exit 1
  fi
  print_success "Messenger linked to profile"

  print_step "2. Storing Telegram token..."
  # Store token in config or environment (simplified here)
  local messenger_config="$HOME/.aps/messengers/${MESSENGER_NAME}"
  if [ -d "$messenger_config" ]; then
    print_info "Token should be stored in: ${messenger_config}/.env"
    echo "TELEGRAM_TOKEN=${TELEGRAM_TOKEN}" > "${messenger_config}/.env.example"
    print_success "Token example stored (use .env for actual token)"
  fi
}

# Step 5: Test and start
final_steps() {
  print_header "Step 5: Ready to Deploy"

  echo -e "${BOLD}Configuration Summary:${RESET}"
  echo "  Profile:     ${CYAN}${PROFILE_ID}${RESET}"
  echo "  Messenger:   ${CYAN}${MESSENGER_NAME}${RESET}"
  echo "  Mappings:    ${CYAN}${#CHANNELS[@]}${RESET} channel(s)"
  if [ -n "$DEFAULT_ACTION" ]; then
    echo "  Default:     ${CYAN}${DEFAULT_ACTION}${RESET}"
  fi
  echo ""

  echo -e "${BOLD}Next steps:${RESET}"
  echo "  1. Configure messenger: nano ${HOME}/.aps/messengers/${MESSENGER_NAME}/config.yaml"
  echo "  2. Start messenger:     ${CYAN}aps messengers start ${MESSENGER_NAME}${RESET}"
  echo "  3. Check status:        ${CYAN}aps messengers status${RESET}"
  echo "  4. View logs:           ${CYAN}aps messengers logs ${MESSENGER_NAME}${RESET}"
  echo ""

  if prompt_yn "Start the messenger now?"; then
    print_info "Starting messenger..."
    if aps messengers start "$MESSENGER_NAME"; then
      print_success "Messenger started"
      print_info "Messenger will begin listening for Telegram messages"
      print_info "Monitor logs with: aps messengers logs ${MESSENGER_NAME} -f"
    else
      print_error "Failed to start messenger"
      echo "Try: aps messengers start ${MESSENGER_NAME} --verbose"
    fi
  fi
}

# Main flow
main() {
  clear
  echo -e "${BOLD}${CYAN}"
  echo "╔═══════════════════════════════════════════════════════════════╗"
  echo "║          APS Telegram Integration Setup                       ║"
  echo "║                                                               ║"
  echo "║  This script will guide you through:                          ║"
  echo "║    1. Creating/selecting an agent profile                     ║"
  echo "║    2. Setting up a Telegram messenger                         ║"
  echo "║    3. Mapping Telegram channels to profile actions            ║"
  echo "║    4. Starting the messenger service                          ║"
  echo "║                                                               ║"
  echo "╚═══════════════════════════════════════════════════════════════╝"
  echo -e "${RESET}"

  check_prerequisites
  setup_profile
  setup_messenger
  setup_channel_mappings
  apply_configuration
  final_steps

  print_header "Setup Complete!"
  print_success "Telegram integration is ready"
  echo ""
}

main "$@"
