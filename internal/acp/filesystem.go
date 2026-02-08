package acp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// MaxFileSize is the maximum file size ACP can read/write (100MB)
	MaxFileSize = 100 * 1024 * 1024

	// SensitivePatterns are patterns of files/paths that should never be accessed
	SensitivePattern1 = ".env"
	SensitivePattern2 = "credentials"
	SensitivePattern3 = "secret"
	SensitivePattern4 = "passwd"
	SensitivePattern5 = "shadow"
)

// FileSystemHandler handles file system operations
type FileSystemHandler struct {
	workingDir string
	maxSize    int64
}

// NewFileSystemHandler creates a new file system handler
func NewFileSystemHandler(workingDir string, maxSize int64) *FileSystemHandler {
	if maxSize <= 0 {
		maxSize = MaxFileSize
	}

	return &FileSystemHandler{
		workingDir: workingDir,
		maxSize:    maxSize,
	}
}

// ReadTextFile reads a text file with security validation
func (fsh *FileSystemHandler) ReadTextFile(path string, startLine int, endLine int) (string, error) {
	// Validate path
	if err := fsh.ValidatePath(path, false); err != nil {
		return "", err
	}

	// Check file size
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", path)
		}
		return "", fmt.Errorf("failed to access file: %w", err)
	}

	if fileInfo.Size() > fsh.maxSize {
		return "", fmt.Errorf("file exceeds maximum size of %d bytes", fsh.maxSize)
	}

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Handle line range selection if provided
	if startLine > 0 || endLine > 0 {
		return selectLines(string(content), startLine, endLine), nil
	}

	return string(content), nil
}

// WriteTextFile writes content to a file with security validation
func (fsh *FileSystemHandler) WriteTextFile(path string, content string) error {
	// Validate path
	if err := fsh.ValidatePath(path, true); err != nil {
		return err
	}

	// Check content size
	if int64(len(content)) > fsh.maxSize {
		return fmt.Errorf("content exceeds maximum size of %d bytes", fsh.maxSize)
	}

	// Create parent directories if needed
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ValidatePath validates a file path for security
func (fsh *FileSystemHandler) ValidatePath(path string, isWrite bool) error {
	// Normalize path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check for path traversal attempts
	if strings.Contains(absPath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Check for sensitive patterns
	if fsh.IsSensitivePath(absPath) {
		return fmt.Errorf("access to this path is not permitted")
	}

	// Check if path is within working directory (if set)
	if fsh.workingDir != "" {
		absWorkDir, err := filepath.Abs(fsh.workingDir)
		if err != nil {
			return fmt.Errorf("invalid working directory: %w", err)
		}

		// Ensure path is within working directory
		if !strings.HasPrefix(absPath, absWorkDir) {
			return fmt.Errorf("access outside working directory not permitted")
		}
	}

	// Additional checks for writes
	if isWrite {
		// Check for system directories
		if isSystemDirectory(absPath) {
			return fmt.Errorf("writing to system directories not permitted")
		}

		// Check if parent directory is writable
		parentDir := filepath.Dir(absPath)
		if err := checkDirWritable(parentDir); err != nil && parentDir != fsh.workingDir {
			return fmt.Errorf("directory is not writable: %w", err)
		}
	}

	return nil
}

// IsSensitivePath checks if a path should not be accessed
func (fsh *FileSystemHandler) IsSensitivePath(path string) bool {
	lowerPath := strings.ToLower(path)

	sensitivePatterns := []string{
		SensitivePattern1,
		SensitivePattern2,
		SensitivePattern3,
		SensitivePattern4,
		SensitivePattern5,
		"/etc/",
		"/sys/",
		"/proc/",
		"/.ssh/",
		"/.aws/",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}

	return false
}

// selectLines returns a subset of lines from content
func selectLines(content string, startLine int, endLine int) string {
	if startLine <= 0 {
		startLine = 1
	}

	lines := strings.Split(content, "\n")

	// Convert to 0-based indices
	startIdx := startLine - 1
	if startIdx >= len(lines) {
		return ""
	}

	if startIdx < 0 {
		startIdx = 0
	}

	endIdx := endLine
	if endLine <= 0 {
		endIdx = len(lines)
	} else {
		endIdx = endLine
	}

	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	return strings.Join(lines[startIdx:endIdx], "\n")
}

// isSystemDirectory checks if a path is a system directory
func isSystemDirectory(path string) bool {
	systemDirs := []string{
		"/bin",
		"/sbin",
		"/boot",
		"/etc",
		"/lib",
		"/lib64",
		"/sys",
		"/proc",
		"/dev",
		"/usr/bin",
		"/usr/sbin",
		"/usr/lib",
	}

	for _, sysDir := range systemDirs {
		if strings.HasPrefix(path, sysDir) {
			return true
		}
	}

	return false
}

// checkDirWritable checks if a directory is writable
func checkDirWritable(dir string) error {
	// Try to create a test file
	testFile := filepath.Join(dir, ".acp_test_write")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return err
	}

	// Clean up test file
	os.Remove(testFile)
	return nil
}

// FileSystemRequest represents a file system request with limits
type FileSystemRequest struct {
	Path      string `json:"path"`
	Content   string `json:"content,omitempty"`
	StartLine int    `json:"startLine,omitempty"`
	EndLine   int    `json:"endLine,omitempty"`
}

// FileSystemResponse represents a file system response
type FileSystemResponse struct {
	Path     string `json:"path"`
	Content  string `json:"content,omitempty"`
	Size     int    `json:"size,omitempty"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}
