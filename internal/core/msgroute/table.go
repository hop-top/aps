package msgroute

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrMissingTerminalRoute is returned when a route table has no terminal
// `match: unknown` fail-safe as its last route.
var ErrMissingTerminalRoute = errors.New("route table must end with a terminal `match: unknown` route")

// Options controls how a Config is compiled into a Table.
type Options struct {
	// Platform is the service adapter (sms, whatsapp, slack, ...). It drives
	// sender key normalization for both inbound senders and table patterns.
	Platform string
	// DefaultProfile fills Route.Profile when a route omits it.
	DefaultProfile string
	// BaseDir resolves relative Config.File and ContactsConfig.Path values.
	// Empty means the process working directory.
	BaseDir string
}

// Table is a compiled, validated route table ready for Resolve.
type Table struct {
	platform string
	source   string
	routes   []compiledRoute
	contacts *contactIndex
}

type matchKind int

const (
	kindSender matchKind = iota
	kindOrg
	kindContact
	kindTerminal
)

type compiledRoute struct {
	route   Route
	kind    matchKind
	pattern string
	glob    bool
}

// Load compiles cfg into a Table, resolving an external file and contacts
// source when declared. All validation problems are aggregated into one
// error (errors.Join) so operators see the whole picture at once.
func Load(cfg *Config, opts Options) (*Table, error) {
	if cfg == nil {
		return nil, errors.New("routing config is required")
	}
	resolved, source, err := resolveConfig(cfg, opts.BaseDir)
	if err != nil {
		return nil, err
	}

	table := &Table{platform: opts.Platform, source: source}
	var problems []error

	if resolved.Contacts != nil {
		index, err := loadContacts(resolved.Contacts, opts)
		if err != nil {
			problems = append(problems, err)
		}
		table.contacts = index
	}

	routes, routeErrs := compileRoutes(resolved.Routes, opts, table.contacts != nil)
	problems = append(problems, routeErrs...)
	table.routes = routes

	if len(problems) > 0 {
		return nil, errors.Join(problems...)
	}
	return table, nil
}

// Routes returns the table's routes in evaluation order with defaulted profiles.
func (t *Table) Routes() []Route {
	if t == nil {
		return nil
	}
	out := make([]Route, 0, len(t.routes))
	for _, r := range t.routes {
		out = append(out, r.route)
	}
	return out
}

// Contacts returns the loaded contact snapshot with normalized keys.
func (t *Table) Contacts() []Contact {
	if t == nil || t.contacts == nil {
		return nil
	}
	return t.contacts.list()
}

// HasTerminal reports whether the table ends with the fail-safe route. A
// Table returned by Load always does; the accessor exists for callers that
// want to assert the invariant.
func (t *Table) HasTerminal() bool {
	if t == nil || len(t.routes) == 0 {
		return false
	}
	return t.routes[len(t.routes)-1].kind == kindTerminal
}

// Source is the external file the table came from, or "inline".
func (t *Table) Source() string {
	if t == nil {
		return ""
	}
	return t.source
}

// Platform is the service platform the table normalizes keys for.
func (t *Table) Platform() string {
	if t == nil {
		return ""
	}
	return t.platform
}

// resolveConfig merges an external routes file into the service-level block.
func resolveConfig(cfg *Config, baseDir string) (*Config, string, error) {
	file := strings.TrimSpace(cfg.File)
	if file == "" {
		return cfg, "inline", nil
	}
	if len(cfg.Routes) > 0 {
		return nil, "", errors.New("routing declares either file or routes, not both")
	}
	resolvedPath, err := resolvePath(file, baseDir)
	if err != nil {
		return nil, "", fmt.Errorf("resolve route table path: %w", err)
	}
	data, err := os.ReadFile(resolvedPath) //nolint:gosec // operator-declared path from service config
	if err != nil {
		return nil, "", fmt.Errorf("read route table %s: %w", resolvedPath, err)
	}
	var external Config
	if err := yaml.Unmarshal(data, &external); err != nil {
		return nil, "", fmt.Errorf("parse route table %s: %w", resolvedPath, err)
	}
	if strings.TrimSpace(external.File) != "" {
		return nil, "", fmt.Errorf("route table %s must not declare file (no nesting)", resolvedPath)
	}
	if external.Contacts != nil && cfg.Contacts != nil {
		return nil, "", fmt.Errorf("contacts declared in both service routing and route table %s", resolvedPath)
	}
	merged := &Config{Contacts: cfg.Contacts, Routes: external.Routes}
	if merged.Contacts == nil && external.Contacts != nil {
		merged.Contacts = anchorContacts(external.Contacts, filepath.Dir(resolvedPath))
	}
	return merged, resolvedPath, nil
}

// anchorContacts re-bases a relative contacts path declared inside an
// external route table onto that table's directory.
func anchorContacts(contacts *ContactsConfig, dir string) *ContactsConfig {
	p := strings.TrimSpace(contacts.Path)
	if p == "" || filepath.IsAbs(p) || strings.HasPrefix(p, "~") {
		return contacts
	}
	copied := *contacts
	copied.Path = filepath.Join(dir, p)
	return &copied
}

func compileRoutes(routes []Route, opts Options, hasContacts bool) ([]compiledRoute, []error) {
	var problems []error
	compiled := make([]compiledRoute, 0, len(routes))
	terminalAt := -1
	for i, route := range routes {
		cr, err := compileRoute(i, route, opts, hasContacts)
		if err != nil {
			problems = append(problems, err)
		}
		if cr.kind == kindTerminal {
			if terminalAt >= 0 {
				problems = append(problems, fmt.Errorf("route %d: duplicate terminal `match: unknown` (first at route %d)", i, terminalAt))
			} else {
				terminalAt = i
			}
		}
		compiled = append(compiled, cr)
	}
	switch {
	case terminalAt < 0:
		problems = append(problems, ErrMissingTerminalRoute)
	case terminalAt != len(routes)-1:
		problems = append(problems, fmt.Errorf("route %d: terminal `match: unknown` must be last; %d unreachable route(s) follow it", terminalAt, len(routes)-1-terminalAt))
	}
	return compiled, problems
}

func compileRoute(index int, route Route, opts Options, hasContacts bool) (compiledRoute, error) {
	var problems []error
	cr := compiledRoute{route: route}

	match := strings.TrimSpace(route.Match)
	if match == "" {
		problems = append(problems, fmt.Errorf("route %d: match is required", index))
	}

	action := strings.TrimSpace(route.Action)
	switch {
	case action == "":
		problems = append(problems, fmt.Errorf("route %d: action is required", index))
	case strings.ContainsAny(action, "= \t"):
		problems = append(problems, fmt.Errorf("route %d: action must be a plain action name, set profile separately (got %q)", index, action))
	}
	cr.route.Action = action

	profile := strings.TrimSpace(route.Profile)
	if profile == "" {
		profile = strings.TrimSpace(opts.DefaultProfile)
	}
	if profile == "" {
		problems = append(problems, fmt.Errorf("route %d: profile is required (no service profile to inherit)", index))
	}
	cr.route.Profile = profile

	if match != "" {
		kind, pattern, err := parseMatch(match, opts.Platform, hasContacts)
		if err != nil {
			problems = append(problems, fmt.Errorf("route %d: %w", index, err))
		}
		cr.kind = kind
		cr.pattern = pattern
		cr.glob = isGlob(pattern)
		if cr.glob {
			if _, err := path.Match(pattern, ""); err != nil {
				problems = append(problems, fmt.Errorf("route %d: invalid glob %q: %w", index, pattern, err))
			}
		}
	}
	return cr, errors.Join(problems...)
}

// parseMatch classifies a match expression and returns its normalized pattern.
func parseMatch(match, platform string, hasContacts bool) (matchKind, string, error) {
	lower := strings.ToLower(match)
	switch {
	case lower == TerminalMatch:
		return kindTerminal, "", nil
	case strings.HasPrefix(lower, orgSelector):
		pattern := strings.ToLower(strings.TrimSpace(match[len(orgSelector):]))
		if pattern == "" {
			return kindOrg, "", errors.New("org selector requires a pattern")
		}
		if !hasContacts {
			return kindOrg, pattern, fmt.Errorf("match %q requires a contacts source", match)
		}
		return kindOrg, pattern, nil
	case strings.HasPrefix(lower, contactSelector):
		pattern := strings.ToLower(strings.TrimSpace(match[len(contactSelector):]))
		if pattern == "" {
			return kindContact, "", errors.New("contact selector requires a pattern")
		}
		if !hasContacts {
			return kindContact, pattern, fmt.Errorf("match %q requires a contacts source", match)
		}
		return kindContact, pattern, nil
	default:
		return kindSender, NormalizeSenderPattern(platform, match), nil
	}
}

func isGlob(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// resolvePath expands a leading ~ and anchors relative paths at baseDir.
func resolvePath(value, baseDir string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "~" || strings.HasPrefix(value, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		value = filepath.Join(home, strings.TrimPrefix(value, "~"))
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value), nil
	}
	if baseDir == "" {
		abs, err := filepath.Abs(value)
		if err != nil {
			return "", fmt.Errorf("resolve %s: %w", value, err)
		}
		return abs, nil
	}
	return filepath.Join(baseDir, value), nil
}
