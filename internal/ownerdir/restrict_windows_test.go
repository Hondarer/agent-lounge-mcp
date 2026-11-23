package ownerdir

import (
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestProtectedOwnerOnlyDACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private")
	if err := Ensure(path); err != nil {
		t.Fatal(err)
	}
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	flags, _, err := sd.Control()
	if err != nil || flags&windows.SE_DACL_PROTECTED == 0 {
		t.Fatalf("DACL is not protected: %v", err)
	}
	acl, _, err := sd.DACL()
	if err != nil || acl.AceCount != 1 {
		t.Fatalf("expected one access entry: %v", err)
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(acl, 0, &ace); err != nil {
		t.Fatal(err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	if !sid.Equals(user.User.Sid) {
		t.Fatal("the DACL grants access to a different user")
	}
}
