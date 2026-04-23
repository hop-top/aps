# Data Model Updates

## Profile (`profile.yaml`)

Update `Preferences` struct:

```yaml
preferences:
  language: string
  timezone: string
  shell: string       # New: Default shell for interactive sessions
```

## Internal Structs

**`internal/core/profile.go`**:

```go
type Preferences struct {
	Language string `yaml:"language,omitempty"`
	Timezone string `yaml:"timezone,omitempty"`
	Shell    string `yaml:"shell,omitempty"` // New
}
```
