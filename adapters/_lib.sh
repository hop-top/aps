# shellcheck shell=bash
# aps adapter shared bash helpers.
#
# Sourced by every script-strategy adapter's backend scripts. Lives
# at adapters/_lib.sh; backend scripts source it via:
#
#   . "$(dirname "$0")/../../../_lib.sh"
#
# Provides:
# - LC_ALL=C pin so external CLIs (gcalcli, gam, himalaya, cardamum)
#   parse dates / numbers reproducibly across operator environments.
# - aps_init_backend <action>: reads the sibling backend manifest at
#   adapters/<adapter>/backends/<backend>/backend.yaml, sets $BIN to
#   the resolved binary path, exits 127 with the manifest-declared
#   install hint if not on PATH.
# - aps_trim_attendee: trims leading/trailing whitespace from a
#   string and extracts the bracketed email from display-name form
#   ("Jane Doe <jane@x.com>" -> "jane@x.com").

LC_ALL="${LC_ALL:-C}"; export LC_ALL

# aps_init_backend pins $BIN to the binary the calling script's
# backend declares in its sibling backend.yaml. Manifest schema:
#
#   binary: gcalcli            # default executable on PATH
#   bin_env_var: GCALCLI_BIN   # operator override env var
#   install_hint: pip install gcalcli
#
# The hint is printed verbatim on a 127 exit so the operator sees
# an actionable next step, not "command not found".
aps_init_backend() {
  local action="$1"
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[1]}")" && pwd)"
  local manifest="$script_dir/backend.yaml"
  if [ ! -f "$manifest" ]; then
    echo "${action}: backend.yaml missing at $manifest" >&2
    exit 78
  fi

  # Flat-YAML parse: grep the three keys, strip the leading "key:"
  # and any surrounding whitespace / quotes. Values may contain
  # spaces, colons, slashes, so we keep everything after the first
  # colon-space split.
  local binary bin_env_var install_hint
  binary="$(aps__yaml_value "$manifest" binary)"
  bin_env_var="$(aps__yaml_value "$manifest" bin_env_var)"
  install_hint="$(aps__yaml_value "$manifest" install_hint)"

  if [ -z "$binary" ]; then
    echo "${action}: backend.yaml missing required 'binary' key ($manifest)" >&2
    exit 78
  fi

  # Operator override env var (bin_env_var) wins over the default
  # binary name. Indirect expansion via ${!var}.
  if [ -n "$bin_env_var" ] && [ -n "${!bin_env_var:-}" ]; then
    BIN="${!bin_env_var}"
  else
    BIN="$binary"
  fi

  command -v "$BIN" >/dev/null 2>&1 || {
    if [ -n "$install_hint" ]; then
      echo "${action}: $binary not installed ($install_hint)" >&2
    else
      echo "${action}: $binary not installed" >&2
    fi
    exit 127
  }
  export BIN
}

# aps__yaml_value extracts a flat top-level scalar from a YAML file.
# Underscore-prefixed = private to this lib. Handles quoted and bare
# values; strips inline comments. Returns empty string if the key is
# absent.
aps__yaml_value() {
  local file="$1"
  local key="$2"
  # 1. Pick the first line matching ^<key>:; 2. strip the leading
  # "key:" up to and including the colon; 3. trim surrounding
  # whitespace; 4. strip matching surrounding quotes; 5. strip
  # trailing inline comments only when the value isn't quoted.
  local raw
  raw=$(grep -E "^${key}:" "$file" | head -1) || true
  if [ -z "$raw" ]; then
    printf ''
    return
  fi
  raw="${raw#"${key}":}"
  raw="${raw#"${raw%%[![:space:]]*}"}"
  raw="${raw%"${raw##*[![:space:]]}"}"
  case "$raw" in
    \"*\") raw="${raw#\"}"; raw="${raw%\"}" ;;
    \'*\') raw="${raw#\'}"; raw="${raw%\'}" ;;
    *\#*) raw="${raw%%#*}"; raw="${raw%"${raw##*[![:space:]]}"}" ;;
  esac
  printf '%s' "$raw"
}

# aps_trim_attendee trims leading/trailing whitespace from $1 and,
# if the result is in display-name form ("Jane Doe <jane@x.com>"),
# returns just the bracketed email. The naive "${a// /}" pattern
# this replaces strips ALL spaces and breaks display-name forms.
aps_trim_attendee() {
  local s="$1"
  s="${s#"${s%%[![:space:]]*}"}"
  s="${s%"${s##*[![:space:]]}"}"
  if [[ "$s" =~ \<([^>]+)\> ]]; then
    s="${BASH_REMATCH[1]}"
  fi
  printf '%s' "$s"
}
