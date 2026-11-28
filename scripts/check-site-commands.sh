#!/usr/bin/env bash
# Verify the `aps ...` command examples on the docs site against the live
# command tree.
#
# Scans fenced bash/sh/shell/zsh blocks in site/src/content/docs/**/*.md(x)
# for lines starting with `aps`. For each line, the command path must be a
# real command (the profile-first shorthand `aps <profile> [-- cmd...]` is
# accepted) and every flag before a `--` separator must exist on that
# command. The valid paths and flags come from `climd --paths`, so they
# cannot drift from the code. Flag values and `<placeholders>` are not
# checked; arguments after a `--` separator are not aps flags.
#
# Limit: `aps <word>` with nothing after it is accepted as the profile-first
# shorthand, so a mistyped single top-level command (`aps wallet`) passes;
# it cannot be told apart from a profile id.
#
# Opt out for a single illustrative line with a trailing comment:
#   aps made up command --x   # docs-check: skip
#
# Env:
#   SITE_CONTENT_DIR   content root (default site/src/content/docs)
#   APS_COMMAND_PATHS  file holding `climd --paths` output (default: run
#                      `go run ./internal/tools/climd --paths`); tests use
#                      this to avoid the Go toolchain
#
# Usage:
#   ./scripts/check-site-commands.sh

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

CONTENT_DIR="${SITE_CONTENT_DIR:-site/src/content/docs}"

if [ -t 1 ]; then
  GREEN='\033[0;32m' RED='\033[0;31m' BOLD='\033[1m' RESET='\033[0m'
else
  GREEN='' RED='' BOLD='' RESET=''
fi

work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT
tree="$work_dir/tree"       # "<path>\t<flags>"
paths="$work_dir/paths"     # "<path>"
examples="$work_dir/examples" # "<file>:<line>\t<text>"

echo -e "${BOLD}Checking site command examples against the command tree...${RESET}"

if [ -n "${APS_COMMAND_PATHS:-}" ]; then
  cp "$APS_COMMAND_PATHS" "$tree"
else
  go run ./internal/tools/climd --paths > "$tree"
fi
cut -f1 "$tree" > "$paths"
if [ ! -s "$paths" ]; then
  echo -e "${RED}command tree is empty; cannot check examples${RESET}" >&2
  exit 1
fi

# Emit "<file>:<line>\t<text>" for each `aps` line inside a shell fence.
extract() {
  awk '
    FNR == 1 { infence = 0; shell = 0 }
    /^[[:space:]]*```/ {
      if (infence) { infence = 0; shell = 0; next }
      infence = 1
      lang = $0; sub(/^[[:space:]]*```/, "", lang); sub(/[[:space:]].*$/, "", lang)
      shell = (lang == "bash" || lang == "sh" || lang == "shell" || lang == "zsh")
      next
    }
    infence && shell {
      line = $0
      sub(/^[[:space:]]+/, "", line)
      sub(/^\$ /, "", line)
      if (line ~ /^aps([[:space:]]|$)/) printf "%s:%d\t%s\n", FILENAME, FNR, line
    }
  ' "$@"
}

files="$(find "$CONTENT_DIR" -type f \( -name '*.md' -o -name '*.mdx' \) | sort)"
if [ -z "$files" ]; then
  echo "no site pages under $CONTENT_DIR"
  exit 0
fi
# shellcheck disable=SC2086 # word-splitting of the file list is intended
extract $files > "$examples"

errors=0
count=0
report() { # <where> <message>
  echo -e "  ${RED}$1${RESET}: $2" >&2
  errors=$((errors + 1))
}

while IFS=$'\t' read -r where text; do
  [ -n "$where" ] || continue
  case "$text" in *"docs-check: skip"*) continue ;; esac
  count=$((count + 1))
  # Drop a trailing shell comment and the [ ] of optional-argument syntax.
  text="${text%% \#*}"
  text="${text//[\[\]]/}"
  read -ra tok <<< "$text"

  # Leading words: the command path plus positional arguments, up to the
  # first flag, placeholder, quoted or `--` token.
  path="aps"
  words=()
  i=1
  while [ "$i" -lt "${#tok[@]}" ]; do
    case "${tok[$i]}" in
      -*|\<*|\"*|\'*|\$*) break ;;
    esac
    words+=("${tok[$i]}")
    i=$((i + 1))
  done
  rest_start="$i"

  # Longest prefix of words that is a command path.
  k=0
  best="aps"
  cand="aps"
  for w in ${words[@]+"${words[@]}"}; do
    cand="$cand $w"
    if grep -qxF -- "$cand" "$paths"; then
      best="$cand"
      k=$((k + 1))
    else
      break
    fi
  done
  path="$best"

  if [ "$k" -eq 0 ]; then
    # Profile-first shorthand: `aps <profile>` or `aps <profile> -- cmd`.
    # Anything else with a leading word is a mistyped command.
    if [ "${#words[@]}" -eq 0 ]; then
      : # `aps`, `aps --flag`, or `aps <placeholder> ...`
    elif [ "${#words[@]}" -eq 1 ] && { [ "$rest_start" -ge "${#tok[@]}" ] || [ "${tok[$rest_start]}" = "--" ]; }; then
      continue # shorthand, nothing aps-side left to check
    else
      report "$where" "unknown command 'aps ${words[0]}' in: $text"
      continue
    fi
  else
    # A group command (has subcommands) followed by a word that is not a
    # subcommand: a mistyped or removed subcommand, not an argument.
    if [ "${#words[@]}" -gt "$k" ] && grep -q -- "^$path " "$paths"; then
      report "$where" "unknown subcommand '${words[$k]}' of '$path' in: $text"
      continue
    fi
  fi

  # Flags: everything up to a bare `--`.
  flags="$(awk -F'\t' -v p="$path" '$1 == p { print " " $2 " " }' "$tree")"
  j="$rest_start"
  while [ "$j" -lt "${#tok[@]}" ]; do
    t="${tok[$j]}"
    j=$((j + 1))
    [ "$t" = "--" ] && break
    case "$t" in
      --*)
        name="${t%%=*}"
        name="${name%%|*}"
        case "$flags" in *" $name "*) ;; *)
          report "$where" "'$path' has no flag $name in: $text" ;;
        esac ;;
      -[A-Za-z]*)
        chars="${t#-}"
        while [ -n "$chars" ]; do
          c="${chars%"${chars#?}"}"
          chars="${chars#?}"
          case "$flags" in *" -$c "*) ;; *)
            report "$where" "'$path' has no flag -$c in: $text" ;;
          esac
        done ;;
    esac
  done
done < "$examples"

if [ "$errors" -gt 0 ]; then
  echo -e "${RED}${BOLD}$errors site command example problem(s).${RESET} Fix the page, or mark an illustrative line with '# docs-check: skip'." >&2
  exit 1
fi
echo -e "${GREEN}All $count site command example(s) match the command tree.${RESET}"
