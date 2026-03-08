package capability

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hop.top/aps/internal/core"

	"gopkg.in/yaml.v3"
)

// TODO: In a real app, inject this via config
func GetCapabilitiesDir() (string, error) {
	usr, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr, ".aps", "capabilities"), nil
}

func GetCapabilityPath(name string) (string, error) {
	root, err := GetCapabilitiesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, name), nil
}

// Install creates a new capability from a source (local copy for now)
func Install(name, source string) error {
	dest, err := GetCapabilityPath(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("capability '%s' already exists", name)
	}

	// For MVP: If source exists locally, copy it. Else just create dir.
	// Real impl would git clone or download.
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	cap := Capability{
		Name:        name,
		Source:      source,
		Path:        dest,
		InstalledAt: time.Now(),
		Type:        TypeManaged,
		Links:       make(map[string]string),
	}

	if source != "" {
		// Simple recursive copy if source is local directory
		if info, err := os.Stat(source); err == nil && info.IsDir() {
			if err := copyDir(source, dest); err != nil {
				return err
			}
		}
	}

	return saveMetadata(cap)
}

// Link creates a symlink to the capability
func Link(name, target string) error {
	cap, err := LoadCapability(name)
	if err != nil {
		return err
	}

	// Resolve target if it's a known tool name (Smart Linking)
	// Resolve target if it's a known tool name (Smart Linking)
	resolvedTarget := target
	// Smart Resolution: if target matches a known tool name (e.g. "windsurf"), use its default path.
	if pattern, err := GetSmartPattern(target); err == nil {
		resolvedTarget = pattern.DefaultPath
	}

	if !filepath.IsAbs(resolvedTarget) {
		cwd, _ := os.Getwd()
		resolvedTarget = filepath.Join(cwd, resolvedTarget)
	}

	// Ensure parent dir exists
	if err := os.MkdirAll(filepath.Dir(resolvedTarget), 0755); err != nil {
		return fmt.Errorf("failed to create parent dir: %w", err)
	}

	// Check if target exists
	if _, err := os.Lstat(resolvedTarget); err == nil {
		return fmt.Errorf("target '%s' already exists", resolvedTarget)
	}

	// Create symlink: Target -> Capability Path
	if err := os.Symlink(cap.Path, resolvedTarget); err != nil {
		return err
	}

	cap.Links[resolvedTarget] = cap.Path
	return saveMetadata(cap)
}

// Delete removes a capability.
// If it's a reference (Watch), it removes the symlink in APS.
// If it's managed (Install/Adopt), it deletes the directory/file in APS.
func Delete(name string) error {
	cap, err := LoadCapability(name)
	if err != nil {
		return err
	}

	// Caller is responsible for warning about breaking links
	_ = cap.Links // checked by callers before calling Delete

	// Safety check: Don't delete root or dangerous paths if something is messed up
	if cap.Path == "" || cap.Path == "/" || cap.Path == "." {
		return fmt.Errorf("invalid capability path: %s", cap.Path)
	}

	if cap.Type == TypeReference {
		// It's a symlink pointing elsewhere
		return os.Remove(cap.Path)
	}

	// It's a directory we own
	return os.RemoveAll(cap.Path)
}

// Adopt moves a file to APS and links it back
func Adopt(target string, name string) error {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}

	if _, err := os.Stat(absTarget); err != nil {
		return fmt.Errorf("target '%s' does not exist", absTarget)
	}

	// Install empty capability
	if err := Install(name, ""); err != nil {
		return err
	}

	cap, err := LoadCapability(name)
	if err != nil {
		return err
	}

	// Move target to capability dir
	baseName := filepath.Base(absTarget)
	destPath := filepath.Join(cap.Path, baseName)
	if err := os.Rename(absTarget, destPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	// Symlink back
	if err := os.Symlink(destPath, absTarget); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	cap.Links[absTarget] = destPath
	return saveMetadata(cap)
}

// Watch links an external file INTO the capability dir
func Watch(target string, name string) error {
	resolvedTarget := target
	// Smart Resolution: if target matches a known tool name (e.g. "windsurf"), use its default path.
	if pattern, err := GetSmartPattern(target); err == nil {
		resolvedTarget = pattern.DefaultPath
	}

	absTarget, err := filepath.Abs(resolvedTarget)
	if err != nil {
		return err
	}

	if _, err := os.Stat(absTarget); err != nil {
		return fmt.Errorf("target '%s' does not exist", absTarget)
	}

	if err := Install(name, ""); err != nil {
		// Use specific error checking?
		// If exists, proceed
	}
	capPath, _ := GetCapabilityPath(name) // Re-fetch or create in-memory?
	// Ensure dir exists
	os.MkdirAll(capPath, 0755)

	destLink := filepath.Join(capPath, filepath.Base(absTarget))
	if err := os.Symlink(absTarget, destLink); err != nil {
		return err
	}

	// Make it a reference type
	cap := Capability{
		Name:        name,
		Path:        capPath,
		InstalledAt: time.Now(),
		Type:        TypeReference,
		Links:       map[string]string{destLink: absTarget},
	}
	return saveMetadata(cap)
}

// Helper to get all source directories (default + configured)
func GetCapabilityRoots() ([]string, error) {
	roots := []string{}

	// Default root
	def, err := GetCapabilitiesDir()
	if err == nil {
		roots = append(roots, def)
	}

	// Configured roots
	cfg, err := core.LoadConfig()
	if err == nil {
		for _, src := range cfg.CapabilitySources {
			if src != "" {
				roots = append(roots, src)
			}
		}
	}

	return roots, nil
}

func LoadCapability(name string) (Capability, error) {
	// Search all roots for the capability
	roots, err := GetCapabilityRoots()
	if err != nil {
		return Capability{}, err
	}

	for _, root := range roots {
		path := filepath.Join(root, name)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return LoadCapabilityFromPath(path, name)
		}
	}

	return Capability{}, fmt.Errorf("capability '%s' not found", name)
}

func LoadCapabilityFromPath(path string, name string) (Capability, error) {
	data, err := os.ReadFile(filepath.Join(path, "manifest.yaml"))
	if err != nil {
		// If no manifest, synthesise one
		if _, statErr := os.Stat(path); statErr == nil {
			return Capability{
				Name:  name,
				Path:  path,
				Type:  TypeManaged,
				Links: make(map[string]string),
			}, nil
		}
		return Capability{}, err
	}

	var cap Capability
	if err := yaml.Unmarshal(data, &cap); err != nil {
		return Capability{}, err
	}
	cap.Path = path

	// Trust manifest if present, else default.
	if cap.Name == "" {
		cap.Name = name
	}
	if cap.Links == nil {
		cap.Links = make(map[string]string)
	}
	return cap, nil
}

func saveMetadata(cap Capability) error {
	data, err := yaml.Marshal(cap)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cap.Path, "manifest.yaml"), data, 0644)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, in)
		return err
	})
}

// List returns all installed capabilities
func List() ([]Capability, error) {
	roots, err := GetCapabilityRoots()
	if err != nil {
		return nil, err
	}

	var caps []Capability
	seen := make(map[string]bool)

	for _, root := range roots {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			if seen[entry.Name()] {
				continue
			}

			cap, err := LoadCapabilityFromPath(filepath.Join(root, entry.Name()), entry.Name())
			if err != nil {
				continue
			}
			caps = append(caps, cap)
			seen[cap.Name] = true
		}
	}
	return caps, nil
}

// GenerateEnvExports generates shell export commands for capabilities
func GenerateEnvExports() ([]string, error) {
	caps, err := List()
	if err != nil {
		return nil, err
	}

	var exports []string

	for _, cap := range caps {
		// key: APS_<NAME>_PATH
		safeName := strings.ToUpper(strings.ReplaceAll(cap.Name, "-", "_"))
		exports = append(exports, fmt.Sprintf("export APS_%s_PATH=\"%s\"", safeName, cap.Path))
	}

	return exports, nil
}
