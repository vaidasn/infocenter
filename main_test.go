package main

import (
	"github.com/rendon/testcli"
	"testing"
)

func init() {
	testcli.Run("go", "install")
}

func TestDefaultInfocenter(t *testing.T) {
	c := testcli.Command("infocenter")
	c.Run()
	if !c.Success() {
		t.Fatalf("Expected to succeed, but failed: %s", c.Error())
	}

	if !c.StdoutContains("port 8080") {
		t.Fatalf("Expected stdout %q to contain %q", c.Stdout(), "port 8080")
	}
}

func TestInfocenterHelp(t *testing.T) {
	c := testcli.Command("infocenter", "--help")
	c.Run()
	if !c.Failure() {
		t.Fatalf("Expected to return non zero return code, but failed: %s", c.Error())
	}

	if c.Stdout() != "" {
		t.Fatalf("Expected stdout %q to be empty", c.Stdout())
	}
	const expectedMessage = "Infocenter server application that uses server-sent events"
	if !c.StderrContains(expectedMessage) {
		t.Fatalf("Expected stderr %q to contain %q", c.Stderr(), expectedMessage)
	}
}
