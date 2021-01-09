// Copyright (c) 2021, Microsoft Corporation, Sean Hinchee
// Licensed under the MIT License.

package cfg

import (
	"os"
	"strings"
	"testing"
)

const (
	testFile = "./test.cfg"
)

// TestLoad tests the cfg.Load method
func TestLoad(t *testing.T) {
	path := testFile
	f, err := os.Open(path)
	if err != nil {
		t.Error("could not open", path, "→", err)
	}

	c, err := Load(f)
	if err != nil {
		t.Error("could not load →", err)
	}

	// Test record count
	if n := len(c.Records); n != 13 {
		t.Error("incorrect record count, got:", n)
	}

	// Test primary keys
	exKeys := []string{`a`, `sys`, `ipnet`, `name`, `creds`, `force`, `c`, `sentence`, `sing`, `quoted`, `test id`, `use bob's code`, `blank`}
	keys := c.Keys()

	if len(exKeys) != len(keys) {
		t.Error("mismatched key lengths")
	}

	for i := 0; i < len(keys); i++ {
		if exKeys[i] != keys[i] {
			t.Error("mismatched primary keys, wanted", exKeys[i], "got", keys[i])
		}
	}
}

// TestMaps checks if Maps() works as intended
func TestMaps(t *testing.T) {
	path := testFile
	f, err := os.Open(path)
	if err != nil {
		t.Error("could not open", path, "→", err)
	}

	c, err := Load(f)
	if err != nil {
		t.Error("could not load →", err)
	}

	creds, ok := c.Lookup("creds")
	if !ok {
		t.Error("Record keyed as 'creds' not found")
	}

	attrs := creds[0].FlatMap()
	exAttrs := []string{`creds`, `username`, `pass`, `method`, `trust`, `known`}

	for _, name := range exAttrs {
		if _, ok := attrs[name]; !ok {
			t.Error("record map is missing name: ", name)
		}
	}
}

// TestEmission checks of what we emit can be loaded back losslessly
func TestEmission(t *testing.T) {
	path := testFile
	f, err := os.Open(path)
	if err != nil {
		t.Error("could not open", path, "→", err)
	}

	c, err := Load(f)
	if err != nil {
		t.Error("could not load →", err)
	}

	var first, second strings.Builder
	c.Emit(&first)
	fs := first.String()

	after, err := Load(strings.NewReader(fs))
	if err != nil {
		t.Error("could not load from first emission")
	}
	after.Emit(&second)
	ss := second.String()

	if fs != ss {
		t.Error("mismatched emissions")
	}
}
