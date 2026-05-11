package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
	kitalias "hop.top/kit/go/console/alias"
)

// ServiceConfig is the persisted profile-facing service definition.
type ServiceConfig struct {
	ID          string            `yaml:"id"`
	Type        string            `yaml:"type"`
	Adapter     string            `yaml:"adapter,omitempty"`
	Profile     string            `yaml:"profile"`
	Description string            `yaml:"description,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Options     map[string]string `yaml:"options,omitempty"`
}

// ServiceRuntimeInfo describes reachable behavior for a persisted service.
type ServiceRuntimeInfo struct {
	Receives string
	Executes string
	Replies  string
	Maturity string
	Routes   []string
}

// ResolvedServiceType records how user-facing service type input resolved.
type ResolvedServiceType struct {
	InputType string
	Type      string
	Adapter   string
	Aliased   bool
}

var serviceTypeAliases = map[string]string{
	"api":      "api agent-protocol",
	"webhook":  "webhook generic",
	"a2a":      "a2a jsonrpc",
	"events":   "events bus",
	"mobile":   "mobile aps",
	"slack":    "message slack",
	"telegram": "message telegram",
	"discord":  "message discord",
	"sms":      "message sms",
	"whatsapp": "message whatsapp",
	"email":    "ticket email",
	"github":   "ticket github",
	"gitlab":   "ticket gitlab",
	"jira":     "ticket jira",
	"linear":   "ticket linear",
}

var canonicalServiceTypes = map[string]bool{
	"api":     true,
	"webhook": true,
	"a2a":     true,
	"client":  true,
	"message": true,
	"ticket":  true,
	"events":  true,
	"mobile":  true,
	"voice":   true,
}

var defaultServiceAdapters = map[string]string{
	"api":     "agent-protocol",
	"webhook": "generic",
	"a2a":     "jsonrpc",
	"events":  "bus",
	"mobile":  "aps",
}

// NewServiceAliasStore returns the kit alias store used for service type
// expansion. Values are encoded as "canonical-type adapter".
func NewServiceAliasStore() *kitalias.Store {
	store := kitalias.NewStore("")
	for name, target := range serviceTypeAliases {
		_ = store.Set(name, target)
	}
	return store
}

// ResolveServiceType expands a canonical service type or adapter alias into
// canonical type/adapter fields.
func ResolveServiceType(typeInput, adapterInput string) (ResolvedServiceType, error) {
	typeInput = strings.TrimSpace(strings.ToLower(typeInput))
	adapterInput = strings.TrimSpace(strings.ToLower(adapterInput))
	if typeInput == "" {
		return ResolvedServiceType{}, fmt.Errorf("service type is required")
	}

	store := NewServiceAliasStore()
	expanded := store.Expand([]string{typeInput})
	if len(expanded) == 2 && expanded[0] != typeInput {
		if adapterInput != "" && adapterInput != expanded[1] {
			return ResolvedServiceType{}, fmt.Errorf(
				"service type alias %q resolves adapter %q, cannot also use adapter %q",
				typeInput, expanded[1], adapterInput,
			)
		}
		return ResolvedServiceType{
			InputType: typeInput,
			Type:      expanded[0],
			Adapter:   expanded[1],
			Aliased:   true,
		}, nil
	}

	if !canonicalServiceTypes[typeInput] {
		return ResolvedServiceType{}, fmt.Errorf("unknown service type or alias %q", typeInput)
	}

	adapter := adapterInput
	if adapter == "" {
		adapter = defaultServiceAdapters[typeInput]
	}
	if adapter == "" {
		return ResolvedServiceType{}, fmt.Errorf("service type %q requires --adapter", typeInput)
	}

	return ResolvedServiceType{
		InputType: typeInput,
		Type:      typeInput,
		Adapter:   adapter,
	}, nil
}

func ServiceTypeAliases() map[string]string {
	out := make(map[string]string, len(serviceTypeAliases))
	for k, v := range serviceTypeAliases {
		out[k] = v
	}
	return out
}

func SortedServiceTypeInputs() []string {
	values := make([]string, 0, len(canonicalServiceTypes)+len(serviceTypeAliases))
	for value := range canonicalServiceTypes {
		values = append(values, value)
	}
	for value := range serviceTypeAliases {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func GetServicesDir() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "services"), nil
}

func GetServicePath(id string) (string, error) {
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("service id cannot be empty")
	}
	dir, err := GetServicesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".yaml"), nil
}

func SaveService(service *ServiceConfig) error {
	if service == nil {
		return fmt.Errorf("service cannot be nil")
	}
	if strings.TrimSpace(service.ID) == "" {
		return fmt.Errorf("service id cannot be empty")
	}
	if strings.TrimSpace(service.Type) == "" {
		return fmt.Errorf("service type cannot be empty")
	}
	if strings.TrimSpace(service.Profile) == "" {
		return fmt.Errorf("service profile cannot be empty")
	}

	path, err := GetServicePath(service.ID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create services directory: %w", err)
	}
	data, err := yaml.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write service: %w", err)
	}
	return nil
}

func LoadService(id string) (*ServiceConfig, error) {
	path, err := GetServicePath(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read service %s: %w", id, err)
	}
	var service ServiceConfig
	if err := yaml.Unmarshal(data, &service); err != nil {
		return nil, fmt.Errorf("failed to parse service %s: %w", id, err)
	}
	if service.ID != id {
		return nil, fmt.Errorf("service ID mismatch: path=%s, content=%s", id, service.ID)
	}
	return &service, nil
}

func DescribeServiceRuntime(service *ServiceConfig) ServiceRuntimeInfo {
	if service == nil {
		return ServiceRuntimeInfo{Maturity: "planned"}
	}

	switch service.Type {
	case "client":
		if service.Adapter == "acp" {
			return ServiceRuntimeInfo{
				Receives: "stdio JSON-RPC",
				Executes: "ACP session, filesystem, terminal, and skill methods",
				Replies:  "JSON-RPC responses",
				Maturity: "ready",
				Routes:   []string{"aps acp server " + service.Profile},
			}
		}
	case "message":
		route := "/services/" + service.ID + "/webhook"
		return ServiceRuntimeInfo{
			Receives: "HTTP POST " + route,
			Executes: "profile action",
			Replies:  service.Adapter + " webhook JSON",
			Maturity: "ready",
			Routes:   []string{route},
		}
	case "ticket":
		return describeTicketServiceRuntime(service)
	}

	return ServiceRuntimeInfo{
		Receives: "not mounted by aps service runtime",
		Executes: "not verified",
		Replies:  "not verified",
		Maturity: "planned",
	}
}

func describeTicketServiceRuntime(service *ServiceConfig) ServiceRuntimeInfo {
	route := "/services/" + service.ID + "/ticket/" + service.Adapter
	info := ServiceRuntimeInfo{
		Receives: "ticket events",
		Executes: "routed profile action with normalized ticket payload",
		Replies:  "status metadata",
		Maturity: "component",
		Routes:   []string{route},
	}
	switch service.Adapter {
	case "jira":
		info.Receives = "Jira issue/comment events"
		info.Replies = "Jira comment body or status metadata"
	case "linear":
		info.Receives = "Linear issue/comment events"
		info.Replies = "Linear comment body or status metadata"
	case "gitlab":
		info.Receives = "GitLab issue/MR/note events"
		info.Replies = "GitLab note body or status metadata"
	}
	return info
}
