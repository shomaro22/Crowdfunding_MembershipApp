package main

import (
	"os"
	"strings"
	"testing"
)

func TestOpenAPIIncludesConnectAndKYCEndpoints(t *testing.T) {
	var (
		data []byte
		err  error
		doc  string
		path string
	)
	data, err = os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	doc = string(data)
	for _, path = range []string{
		"/seller/connect:",
		"/seller/connect/onboarding:",
		"/admin/kyc-documents:",
		"ConnectAccountResponse:",
		"ConnectOnboardingResponse:",
	} {
		if !strings.Contains(doc, path) {
			t.Fatalf("openapi.yaml missing %q", path)
		}
	}
}
