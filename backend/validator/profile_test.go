package validator

import (
	"strings"
	"testing"

	"crowdfund-app/model"
)

func TestValidateUpdateProfile(t *testing.T) {
	var (
		name  = "Alice"
		avatar = "https://example.com/a.png"
		out   UpdateProfileNormalized
		err   error
	)
	out, err = ValidateUpdateProfile(UpdateProfileInput{Name: &name})
	if err != nil || !out.NameSet || out.Name == nil || *out.Name != "Alice" {
		t.Fatalf("name update: err=%v out=%+v", err, out)
	}

	out, err = ValidateUpdateProfile(UpdateProfileInput{AvatarURL: &avatar})
	if err != nil || !out.AvatarURLSet || out.AvatarURL == nil || *out.AvatarURL != avatar {
		t.Fatalf("avatar update: err=%v out=%+v", err, out)
	}

	_, err = ValidateUpdateProfile(UpdateProfileInput{})
	if err == nil {
		t.Fatal("expected error for empty update")
	}

	bad := "ftp://example.com/a.png"
	_, err = ValidateUpdateProfile(UpdateProfileInput{AvatarURL: &bad})
	if err == nil {
		t.Fatal("expected error for ftp avatar url")
	}

	longName := strings.Repeat("a", MaxUserNameLen+1)
	_, err = ValidateUpdateProfile(UpdateProfileInput{Name: &longName})
	if err != model.ErrBadProfile {
		t.Fatalf("long name err=%v", err)
	}

	empty := ""
	out, err = ValidateUpdateProfile(UpdateProfileInput{Name: &empty})
	if err != nil || out.Name != nil {
		t.Fatalf("clear name: err=%v name=%v", err, out.Name)
	}
}
