# shellcheck shell=bash
# aps adapter shared bash helpers.
#
# Sourced by every script-strategy adapter's backend scripts. Lives
# at adapters/_lib.sh; backend scripts source it via:
#
#   . "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../_lib.sh"
#
# The cd+pwd dance resolves symlinks so the path holds even when the
# dispatcher (or an operator) invokes the script through a symlink.
# The three-up `../../../` segment requires every adapter script to
# live exactly at adapters/<adapter>/backends/<backend>/<action>.sh —
# the path breaks for any deeper layout.
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
# backend declares in its sibling backend.yaml. Manifest schema is
# flat inline YAML scalars (block scalars > / | / [ are rejected):
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
  if [ ! -r "$manifest" ]; then
    echo "${action}: backend.yaml not readable at $manifest (check file permissions)" >&2
    exit 78
  fi

  local binary bin_env_var install_hint
  binary="$(aps__yaml_value "$action" "$manifest" binary)"
  bin_env_var="$(aps__yaml_value "$action" "$manifest" bin_env_var)"
  install_hint="$(aps__yaml_value "$action" "$manifest" install_hint)"

  if [ -z "$binary" ]; then
    echo "${action}: backend.yaml missing required 'binary' key ($manifest)" >&2
    exit 78
  fi

  # Operator override env var (bin_env_var) wins over the default
  # binary name. Indirect expansion via ${!var}: empty / unset env
  # falls through to the default safely.
  if [ -n "$bin_env_var" ] && [ -n "${!bin_env_var:-}" ]; then
    BIN="${!bin_env_var}"
  else
    BIN="$binary"
  fi

  # command -v finds anything on PATH including non-executable files
  # and directories. Resolve then assert -x so a directory named
  # `gcalcli` or a non-executable shim surfaces here instead of
  # downstream as "cannot execute: Is a directory".
  local resolved
  resolved="$(command -v "$BIN" 2>/dev/null || true)"
  if [ -z "$resolved" ] || [ ! -x "$resolved" ]; then
    if [ -n "$install_hint" ]; then
      echo "${action}: $binary not installed or not executable ($install_hint)" >&2
    else
      echo "${action}: $binary not installed or not executable" >&2
    fi
    exit 127
  fi
  export BIN
}

# aps__yaml_value extracts a flat top-level inline scalar from a YAML
# file. Underscore-prefixed = private to this lib. Handles quoted
# (single + double) and bare values; strips inline comments on bare
# values; strips a trailing CR for CRLF-edited manifests; rejects
# YAML block-scalar markers (>, |, [) with an explicit error so a
# folded/literal/flow value never silently returns its indicator
# character. Warns on duplicate top-level keys.
aps__yaml_value() {
  local action="$1"
  local file="$2"
  local key="$3"
  local all
  all=$(grep -E "^${key}:" "$file" 2>/dev/null) || true
  if [ -z "$all" ]; then
    printf ''
    return
  fi

  local match_count
  match_count=$(printf '%s\n' "$all" | wc -l | tr -d ' ')
  if [ "$match_count" -gt 1 ]; then
    echo "${action}: warning: backend.yaml has $match_count entries for '$key'; using first" >&2
  fi

  local raw
  raw=$(printf '%s\n' "$all" | head -1)
  raw="${raw%$'\r'}"
  raw="${raw#"${key}":}"
  raw="${raw#"${raw%%[![:space:]]*}"}"
  raw="${raw%"${raw##*[![:space:]]}"}"

  case "$raw" in
    \>*|\|*|\[*|\{*)
      echo "${action}: backend.yaml '$key' uses unsupported block/flow scalar ('${raw:0:1}'); use an inline string instead" >&2
      exit 78
      ;;
    \"*\") raw="${raw#\"}"; raw="${raw%\"}" ;;
    \'*\') raw="${raw#\'}"; raw="${raw%\'}" ;;
    *\#*) raw="${raw%%#*}"; raw="${raw%"${raw##*[![:space:]]}"}" ;;
  esac
  printf '%s' "$raw"
}

# aps_sed_replacement escapes $1 for safe use on the RIGHT-hand side
# of a sed `s///` substitution.
#
# Three characters are special there and must be backslash-escaped:
#   /  closes the replacement early — the remainder is then parsed as
#      sed flags ("bad flag in substitute command")
#   &  expands to the entire matched text, so "R&D" silently becomes
#      "R<whole matched line>D"
#   \  introduces an escape sequence
#
# All three are ordinary in contact fields: job titles ("Eng/Ops"),
# organisation names ("R&D"), addresses, and URLs in notes. Callers
# interpolating an untrusted value into `s/^KEY:.*/KEY:$VALUE/` must
# route it through this first.
aps_sed_replacement() {
  printf '%s' "$1" | sed -e 's/[&\\/]/\\&/g'
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
