//go:build windows

package fsperm

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Well-known SIDs always allowed on a secret file.
const (
	sidSystem = "S-1-5-18"     // SYSTEM
	sidAdmins = "S-1-5-32-544" // BUILTIN\Administrators
)

func currentUserSID() (string, error) {
	u, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", fmt.Errorf("fsperm: token user: %w", err)
	}
	return u.User.Sid.String(), nil
}

func openSecret(path string, flag int, perm os.FileMode) (*os.File, error) {
	f, err := os.OpenFile(path, flag, perm) // #nosec G304 -- caller-controlled secret path
	if err != nil {
		return nil, err //nolint:wrapcheck // PathError passthrough
	}
	// Restrict before any data is written; a freshly created file is empty.
	if err := protect(path); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

// protect replaces path's DACL with a protected (non-inheriting) DACL granting
// full control to the current user, SYSTEM and Administrators only. The token
// user SID is used rather than the OW (owner rights) alias because a process
// running elevated creates files owned by Administrators, not the user.
func protect(path string) error {
	user, err := currentUserSID()
	if err != nil {
		return err
	}
	sddl := fmt.Sprintf("D:P(A;;FA;;;%s)(A;;FA;;;%s)(A;;FA;;;%s)", user, sidSystem, sidAdmins)
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return fmt.Errorf("fsperm: build descriptor: %w", err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return fmt.Errorf("fsperm: descriptor DACL: %w", err)
	}
	err = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, dacl, nil)
	if err != nil {
		return fmt.Errorf("fsperm: set DACL on %s: %w", path, err)
	}
	return nil
}

func checkSecret(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", err //nolint:wrapcheck // PathError passthrough
	}
	user, err := currentUserSID()
	if err != nil {
		return "", err
	}
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return "", fmt.Errorf("fsperm: read DACL of %s: %w", path, err)
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return "", fmt.Errorf("fsperm: descriptor DACL: %w", err)
	}
	if dacl == nil {
		// NULL DACL grants everyone full access.
		return "insecure permissions (no DACL: everyone has access). It should be restricted to the owner, SYSTEM and Administrators.", nil
	}
	allowed := map[string]bool{user: true, sidSystem: true, sidAdmins: true}
	var extra []string
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return "", fmt.Errorf("fsperm: read ACE %d of %s: %w", i, path, err)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			continue // deny and audit ACEs grant nothing
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		if !allowed[sid] {
			extra = append(extra, sid)
		}
	}
	if len(extra) > 0 {
		return fmt.Sprintf("insecure permissions (ACL grants access to %s). It should be restricted to the owner, SYSTEM and Administrators.",
			strings.Join(extra, ", ")), nil
	}
	return "", nil
}
