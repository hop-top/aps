package acp

import (
	"os"
	"path/filepath"
	"testing"

	"oss-aps-cli/internal/acp"
)

// TestFileSystemHandlerRead tests reading a file
func TestFileSystemHandlerRead(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line 1\nline 2\nline 3\n"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	fsh := acp.NewFileSystemHandler(tmpDir, acp.MaxFileSize)

	// Read the file
	content, err := fsh.ReadTextFile(testFile, 0, 0)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if content != testContent {
		t.Errorf("content mismatch: expected '%s', got '%s'", testContent, content)
	}
}

// TestFileSystemHandlerWrite tests writing a file
func TestFileSystemHandlerWrite(t *testing.T) {
	tmpDir := t.TempDir()

	fsh := acp.NewFileSystemHandler(tmpDir, acp.MaxFileSize)

	testFile := filepath.Join(tmpDir, "output.txt")
	testContent := "test content"

	if err := fsh.WriteTextFile(testFile, testContent); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("written content mismatch: expected '%s', got '%s'", testContent, string(content))
	}
}

// TestFileSystemHandlerPathTraversal tests path traversal prevention
func TestFileSystemHandlerPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	fsh := acp.NewFileSystemHandler(tmpDir, acp.MaxFileSize)

	// Try to read a file outside the working directory using path traversal
	maliciousPath := filepath.Join(tmpDir, "..", "..", "etc", "passwd")

	_, err := fsh.ReadTextFile(maliciousPath, 0, 0)
	if err == nil {
		t.Error("should deny path traversal attempts")
	}
}

// TestFileSystemHandlerSensitivePaths tests blocking sensitive paths
func TestFileSystemHandlerSensitivePaths(t *testing.T) {
	tmpDir := t.TempDir()
	fsh := acp.NewFileSystemHandler(tmpDir, acp.MaxFileSize)

	sensitiveFiles := []string{
		filepath.Join(tmpDir, ".env"),
		filepath.Join(tmpDir, "credentials.json"),
		filepath.Join(tmpDir, "secret_key.txt"),
	}

	for _, sensitiveFile := range sensitiveFiles {
		_, err := fsh.ReadTextFile(sensitiveFile, 0, 0)
		if err == nil {
			t.Errorf("should deny access to sensitive file: %s", sensitiveFile)
		}
	}
}

// TestFileSystemHandlerMaxSize tests file size limits
func TestFileSystemHandlerMaxSize(t *testing.T) {
	tmpDir := t.TempDir()
	maxSize := int64(100) // 100 bytes

	fsh := acp.NewFileSystemHandler(tmpDir, maxSize)

	// Create a file larger than max size
	largeFile := filepath.Join(tmpDir, "large.txt")
	largeContent := make([]byte, maxSize+1)
	for i := range largeContent {
		largeContent[i] = 'a'
	}

	if err := os.WriteFile(largeFile, largeContent, 0644); err != nil {
		t.Fatalf("failed to create large file: %v", err)
	}

	// Try to read it
	_, err := fsh.ReadTextFile(largeFile, 0, 0)
	if err == nil {
		t.Error("should deny reading files larger than max size")
	}
}

// TestFileSystemHandlerWriteMaxSize tests write size limits
func TestFileSystemHandlerWriteMaxSize(t *testing.T) {
	tmpDir := t.TempDir()
	maxSize := int64(100)

	fsh := acp.NewFileSystemHandler(tmpDir, maxSize)

	// Try to write content larger than max size
	largeContent := make([]byte, maxSize+1)
	for i := range largeContent {
		largeContent[i] = 'a'
	}

	err := fsh.WriteTextFile(filepath.Join(tmpDir, "output.txt"), string(largeContent))
	if err == nil {
		t.Error("should deny writing files larger than max size")
	}
}

// TestFileSystemHandlerLineSelection tests line selection
func TestFileSystemHandlerLineSelection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "lines.txt")
	testContent := "line 1\nline 2\nline 3\nline 4\n"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	fsh := acp.NewFileSystemHandler(tmpDir, acp.MaxFileSize)

	// Read lines 2-3
	content, err := fsh.ReadTextFile(testFile, 2, 3)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	expected := "line 2\nline 3"
	if content != expected {
		t.Errorf("line selection mismatch: expected '%s', got '%s'", expected, content)
	}
}

// TestIsSensitivePath tests sensitive path detection
func TestIsSensitivePath(t *testing.T) {
	tmpDir := t.TempDir()
	fsh := acp.NewFileSystemHandler(tmpDir, acp.MaxFileSize)

	testCases := []struct {
		path       string
		sensitive  bool
	}{
		{"/etc/passwd", true},
		{"/home/user/.env", true},
		{"/home/user/credentials", true},
		{"/home/user/secret.txt", true},
		{"/home/user/document.txt", false},
		{"/tmp/file.txt", false},
	}

	for _, tc := range testCases {
		result := fsh.IsSensitivePath(tc.path)
		if result != tc.sensitive {
			t.Errorf("IsSensitivePath(%s): expected %v, got %v", tc.path, tc.sensitive, result)
		}
	}
}

// TestValidatePath tests path validation
func TestValidatePath(t *testing.T) {
	tmpDir := t.TempDir()
	fsh := acp.NewFileSystemHandler(tmpDir, acp.MaxFileSize)

	// Valid path within working directory
	validPath := filepath.Join(tmpDir, "file.txt")
	if err := fsh.ValidatePath(validPath, false); err != nil {
		t.Errorf("should allow valid path: %v", err)
	}

	// Path outside working directory
	outsidePath := "/etc/passwd"
	if err := fsh.ValidatePath(outsidePath, false); err == nil {
		t.Error("should deny paths outside working directory")
	}
}
