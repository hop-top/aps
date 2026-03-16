#!/usr/bin/env bash
# Generic interactive setup script for any APS messenger
#
# Guides users through configuring any messenger (Telegram, Slack, GitHub, Email)
# with an agent profile and channel mappings.
#
# Usage:
#   ./scripts/setup-messenger.sh              # interactive
#   ./scripts/setup-messenger.sh --type=telegram
#   ./scripts/setup-messenger.sh --type=slack --profile=my-profile

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
MESSENGER_TYPE=""
MESSENGER_NAME=""
PROFILE_ID=""
CHANNELS=()
DEFAULT_ACTION=""
SKIP_CONFIRMATION=false

# Supported messenger platforms
declare -A PLATFORM_INFO=(
  [telegram]="Telegram|numeric chat ID"
  [slack]="Slack|channel ID (C...)"
  [discord]="Discord|numeric channel ID"
  [github]="GitHub|org/repo"
  [email]="Email|mailbox or address"
)

# Helper functions
print_header() {
  echo -e "\n${BOLD}${CYAN}==== $1 ====${RESET}\n"
}

print_step() {
  echo -e "${BOLD}$1${RESET}"
}

print_info() {
  echo -e "${CYAN}ℹ${RESET}  $1"
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

# Parse command line arguments
parse_args() {
  while [ $# -gt 0 ]; do
    case "$1" in
      --type=*)
        MESSENGER_TYPE="${1#*=}"
        ;;
      --profile=*)
        PROFILE_ID="${1#*=}"
        ;;
      --messenger=*)
        MESSENGER_NAME="${1#*=}"
        ;;
      --yes|-y)
        SKIP_CONFIRMATION=true
        ;;
      --help|-h)
        show_help
        exit 0
        ;;
      *)
        print_error "Unknown option: $1"
        show_help
        exit 1
        ;;
    esac
    shift
  done
}

show_help() {
  cat << EOF
${BOLD}APS Messenger Setup Script${RESET}

Setup interactive configuration for any APS messenger platform.

${BOLD}Usage:${RESET}
  ./scripts/setup-messenger.sh [OPTIONS]

${BOLD}Options:${RESET}
  --type=TYPE              Messenger type: telegram, slack, github, email
  --profile=PROFILE        Profile ID (skip profile selection step)
  --messenger=NAME         Messenger name (skip naming step)
  --yes, -y                Skip confirmation prompts
  --help, -h               Show this help

${BOLD}Supported Messengers:${RESET}
  • telegram  - Telegram bot integration
  • slack     - Slack workspace integration
  • discord   - Discord server integration
  • github    - GitHub webhook integration
  • email     - Email gateway integration

${BOLD}Examples:${RESET}
  # Interactive setup for Telegram
  ./scripts/setup-messenger.sh --type=telegram

  # Quick setup for Discord
  ./scripts/setup-messenger.sh --type=discord --profile=my-bot --messenger=my-discord

  # Quick setup for Slack with auto-confirmation
  ./scripts/setup-messenger.sh --type=slack --profile=my-bot --messenger=my-slack --yes

  # Setup with auto-confirmation
  ./scripts/setup-messenger.sh --type=github --yes

EOF
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
}

# Select or determine messenger type
select_messenger_type() {
  if [ -n "$MESSENGER_TYPE" ]; then
    return
  fi

  print_header "Messenger Type Selection"
  echo "Supported messengers:"
  echo "  1. telegram  - Telegram bot"
  echo "  2. slack     - Slack workspace"
  echo "  3. discord   - Discord server"
  echo "  4. github    - GitHub webhooks"
  echo "  5. email     - Email gateway"
  echo ""

  while true; do
    choice=$(prompt "Select messenger type (1-5 or name)")
    case "$choice" in
      1|telegram)
        MESSENGER_TYPE="telegram"
        break
        ;;
      2|slack)
        MESSENGER_TYPE="slack"
        break
        ;;
      3|discord)
        MESSENGER_TYPE="discord"
        break
        ;;
      4|github)
        MESSENGER_TYPE="github"
        break
        ;;
      5|email)
        MESSENGER_TYPE="email"
        break
        ;;
      *)
        print_warning "Invalid selection"
        ;;
    esac
  done

  print_success "Messenger type: ${CYAN}${MESSENGER_TYPE}${RESET}"
}

# Setup profile
setup_profile() {
  if [ -n "$PROFILE_ID" ]; then
    print_info "Using profile: ${CYAN}${PROFILE_ID}${RESET}"
    return
  fi

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
    fi
  else
    PROFILE_ID=$(prompt "Create new profile with ID")
  fi

  # Create profile if needed
  if ! aps profile get "$PROFILE_ID" &>/dev/null; then
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

# Setup messenger
setup_messenger() {
  print_header "Step 2: Messenger Setup"

  if [ -z "$MESSENGER_NAME" ]; then
    local default_name="${MESSENGER_TYPE}-$(date +%s | tail -c 5)"
    MESSENGER_NAME=$(prompt "Messenger name" "$default_name")
  fi

  print_success "Messenger name: ${CYAN}${MESSENGER_NAME}${RESET}"

  # Check if messenger exists
  if [ -f "$HOME/.aps/messengers/${MESSENGER_NAME}/config.yaml" ]; then
    print_warning "Messenger '${MESSENGER_NAME}' already exists"
    if ! $SKIP_CONFIRMATION && ! prompt_yn "Configure existing messenger?"; then
      MESSENGER_NAME=$(prompt "New messenger name")
    fi
  fi

  # Decide on mode
  print_info "Deployment mode:"
  local mode
  if $SKIP_CONFIRMATION; then
    mode="subprocess"
  else
    if prompt_yn "Subprocess mode (always listening)?"; then
      mode="subprocess"
    else
      mode="webhook"
    fi
  fi

  # Language selection
  local language
  if $SKIP_CONFIRMATION; then
    language="python"
  else
    language=$(prompt "Implementation language" "python")
  fi

  print_info "Creating ${MESSENGER_TYPE} messenger (${language}, ${mode} mode)..."
  if ! aps messengers create "$MESSENGER_NAME" \
    --template="$mode" \
    --language="$language" 2>/dev/null; then
    print_error "Failed to create messenger"
    exit 1
  fi
  print_success "Messenger created"
}

# Setup credentials
setup_credentials() {
  print_header "Step 3: Credentials"

  case "$MESSENGER_TYPE" in
    telegram)
      print_info "You need a Telegram bot token from BotFather:"
      echo "  1. Open Telegram and search for @BotFather"
      echo "  2. Send /newbot and follow the steps"
      echo "  3. Copy the bot token"
      echo ""
      local token
      token=$(prompt "Telegram bot token")
      echo "TELEGRAM_TOKEN=${token}" > "$HOME/.aps/messengers/${MESSENGER_NAME}/.env"
      print_success "Token stored"
      ;;
    slack)
      print_info "You need a Slack bot token:"
      echo "  1. Go to https://api.slack.com/apps"
      echo "  2. Create or select your app"
      echo "  3. Get the Bot User OAuth Token"
      echo ""
      local token
      token=$(prompt "Slack bot token")
      echo "SLACK_TOKEN=${token}" > "$HOME/.aps/messengers/${MESSENGER_NAME}/.env"
      print_success "Token stored"
      ;;
    discord)
      print_info "You need a Discord bot token:"
      echo "  1. Go to https://discord.com/developers/applications"
      echo "  2. Create or select your app"
      echo "  3. Go to Bot section and copy the token"
      echo ""
      local token
      token=$(prompt "Discord bot token")
      echo "DISCORD_TOKEN=${token}" > "$HOME/.aps/messengers/${MESSENGER_NAME}/.env"
      print_success "Token stored"
      ;;
    github)
      print_info "GitHub integration uses webhooks"
      print_info "Configure webhook in repository settings → Webhooks"
      ;;
    email)
      print_info "Email integration configuration"
      local smtp_host
      smtp_host=$(prompt "SMTP host" "smtp.gmail.com")
      local smtp_port
      smtp_port=$(prompt "SMTP port" "587")
      echo "SMTP_HOST=${smtp_host}" > "$HOME/.aps/messengers/${MESSENGER_NAME}/.env"
      echo "SMTP_PORT=${smtp_port}" >> "$HOME/.aps/messengers/${MESSENGER_NAME}/.env"
      print_success "Email config stored"
      ;;
  esac

  chmod 600 "$HOME/.aps/messengers/${MESSENGER_NAME}/.env"
}

# Setup channel mappings
setup_channel_mappings() {
  print_header "Step 4: Configure Mappings"

  local platform_info=${PLATFORM_INFO[$MESSENGER_TYPE]:-"$MESSENGER_TYPE|unknown"}
  IFS='|' read -r platform_name channel_format <<< "$platform_info"

  print_info "Map ${platform_name} channels to profile actions"
  echo "  Channel ID format: $channel_format"
  echo ""

  while true; do
    local channel_id
    channel_id=$(prompt "Channel ID (or 'done' to finish)")

    if [ "$channel_id" = "done" ] || [ "$channel_id" = "Done" ]; then
      break
    fi

    if [ -z "$channel_id" ]; then
      print_warning "Channel ID cannot be empty"
      continue
    fi

    local action
    action=$(prompt "Action to execute")

    if [ -z "$action" ]; then
      print_warning "Action cannot be empty"
      continue
    fi

    local mapping="${PROFILE_ID}=${action}"
    CHANNELS+=("${channel_id}:${mapping}")
    print_success "Mapped: ${CYAN}${channel_id}${RESET} → ${CYAN}${action}${RESET}"
  done

  if [ ${#CHANNELS[@]} -eq 0 ]; then
    print_warning "No mappings configured"
    if prompt_yn "Set a default action instead?"; then
      DEFAULT_ACTION=$(prompt "Default action")
    fi
  else
    echo ""
    if prompt_yn "Set default action for unmapped channels?"; then
      DEFAULT_ACTION=$(prompt "Default action")
    fi
  fi
}

# Apply configuration
apply_configuration() {
  print_header "Step 5: Applying Configuration"

  print_step "Linking messenger to profile..."

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
    print_error "Failed to link messenger"
    exit 1
  fi
  print_success "Messenger linked to profile"
}

# Final steps
final_steps() {
  print_header "Configuration Complete"

  echo -e "${BOLD}Summary:${RESET}"
  echo "  Type:        ${CYAN}${MESSENGER_TYPE}${RESET}"
  echo "  Messenger:   ${CYAN}${MESSENGER_NAME}${RESET}"
  echo "  Profile:     ${CYAN}${PROFILE_ID}${RESET}"
  echo "  Mappings:    ${CYAN}${#CHANNELS[@]}${RESET} channel(s)"
  if [ -n "$DEFAULT_ACTION" ]; then
    echo "  Default:     ${CYAN}${DEFAULT_ACTION}${RESET}"
  fi
  echo ""

  echo -e "${BOLD}Next steps:${RESET}"
  echo "  1. Review config:   cat ~/.aps/messengers/${MESSENGER_NAME}/config.yaml"
  echo "  2. Start messenger: ${CYAN}aps messengers start ${MESSENGER_NAME}${RESET}"
  echo "  3. Check status:    ${CYAN}aps messengers status${RESET}"
  echo "  4. View logs:       ${CYAN}aps messengers logs ${MESSENGER_NAME} -f${RESET}"
  echo ""

  if $SKIP_CONFIRMATION || prompt_yn "Start messenger now?"; then
    print_info "Starting..."
    if aps messengers start "$MESSENGER_NAME"; then
      print_success "Messenger started"
    else
      print_error "Failed to start (try: aps messengers start ${MESSENGER_NAME} --verbose)"
    fi
  fi
}

# Main
main() {
  parse_args "$@"

  clear
  echo -e "${BOLD}${CYAN}"
  echo "╔═══════════════════════════════════════════════════════════════╗"
  echo "║              APS Messenger Setup                              ║"
  echo "║                                                               ║"
  echo "║  Configure any messenger (Telegram, Slack, GitHub, Email)     ║"
  echo "║  with an agent profile and channel mappings.                  ║"
  echo "║                                                               ║"
  echo "╚═══════════════════════════════════════════════════════════════╝"
  echo -e "${RESET}"

  check_prerequisites
  select_messenger_type
  setup_profile
  setup_messenger
  setup_credentials
  setup_channel_mappings
  apply_configuration
  final_steps

  print_header "Ready to Deploy"
  print_success "Setup complete - messenger is live"
  echo ""
}

main "$@"
