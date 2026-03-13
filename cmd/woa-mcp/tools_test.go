package main

import "testing"

func TestBuildMCPServer_NotNil(t *testing.T) {
	mc := newMockClient()
	buf := newEventBuf(100)
	s := buildMCPServer(mc, buf)
	if s == nil {
		t.Fatal("buildMCPServer returned nil")
	}
}
