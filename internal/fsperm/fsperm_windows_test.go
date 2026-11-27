//go:build windows

package fsperm

import (
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func readACL(t *testing.T, path string) (sids []string, protected bool) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	ctrl, _, err := sd.Control()
	if err != nil {
		t.Fatal(err)
	}
	protected = ctrl&windows.SE_DACL_PROTECTED != 0
	dacl, _, err := sd.DACL()
	if err != nil || dacl == nil {
		t.Fatalf("dacl = %v, %v", dacl, err)
	}
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			t.Fatal(err)
		}
		sids = append(sids, (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String())
	}
	return sids, protected
}

func TestWindowsDACLOwnerOnly(t *testing.T) {
	p := filepath.Join(t.TempDir(), "secrets.env")
	if err := WriteSecret(p, []byte("K=V\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	user, err := currentUserSID()
	if err != nil {
		t.Fatal(err)
	}
	sids, protected := readACL(t, p)
	if !protected {
		t.Error("DACL not protected: parent ACEs still inherited")
	}
	want := map[string]bool{user: false, sidSystem: false, sidAdmins: false}
	for _, s := range sids {
		if _, ok := want[s]; !ok {
			t.Errorf("unexpected principal %s in DACL %v", s, sids)
			continue
		}
		want[s] = true
	}
	for s, seen := range want {
		if !seen {
			t.Errorf("missing ACE for %s in %v", s, sids)
		}
	}
	if warn, err := CheckSecret(p); err != nil || warn != "" {
		t.Errorf("CheckSecret = %q, %v", warn, err)
	}
}

func TestWindowsCheckDetectsEveryone(t *testing.T) {
	p := filepath.Join(t.TempDir(), "secrets.env")
	if err := WriteSecret(p, []byte("K=V\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;WD)")
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, _ := sd.DACL()
	if err := windows.SetNamedSecurityInfo(p, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
	warn, err := CheckSecret(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warn, "S-1-1-0") {
		t.Fatalf("warn = %q, want Everyone SID", warn)
	}
	// Rewriting re-protects.
	if err := WriteSecret(p, []byte("K=V\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if warn, _ := CheckSecret(p); warn != "" {
		t.Fatalf("not re-protected: %q", warn)
	}
}
