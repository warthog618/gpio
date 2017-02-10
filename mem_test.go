/*
  Test suite for mem module.

*/
package gpio

import (
	"testing"
)

func TestOpen(t *testing.T) {
	if err := Open(); err != nil {
		t.Fatal("Open returned error", err)
	}
	defer Close()
}

func TestOpenOpened(t *testing.T) {
	if err := Open(); err != nil {
		t.Fatal("Open returned error", err)
	}
	defer Close()
	if err := Open(); err == nil {
		t.Fatal("Open when opened didn't return error")
	}
}

func TestReOpen(t *testing.T) {
	if err := Open(); err != nil {
		t.Fatal("Open returned error", err)
	}
	Close()
	if err := Open(); err != nil {
		t.Fatal("Open returned error", err)
	}
	defer Close()
}
