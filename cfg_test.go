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

// TestFlatMap checks if FlatMap works as intended
func TestFlatMap(t *testing.T) {
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

	// Flatmap
	attrs := creds[0].FlatMap()
	exAttrs := []string{`creds`, `username`, `pass`, `method`, `trust`, `known`}

	for _, name := range exAttrs {
		if _, ok := attrs[name]; !ok {
			t.Error("record map is missing name: ", name)
		}
	}
}

// TestMap checks if .Map works as intended
func TestMap(t *testing.T) {
	path := testFile
	f, err := os.Open(path)
	if err != nil {
		t.Error("could not open", path, "→", err)
	}

	c, err := Load(f)
	if err != nil {
		t.Error("could not load →", err)
	}

	// Test deeply nested lookup
	authdom, ok := c.Map["ipnet"]["auth"]["authdom"]
	if !ok {
		t.Error("ipnet → auth → authdom not found in map")
	}

	if len(authdom) < 1 || authdom[0] != "HOME" {
		t.Error("incorrect value in map for authdom")
	}

	// Test valueless singleton names
	first, ok := c.Map["blank"]["first"]
	if !ok {
		t.Error("blank → first tuple not found in map")
	}

	names := make(map[string]int)
	exNames := []string{"first", "second", "third", "fourth", "fifth's sixth"}
	for name, value := range first {
		if len(value) > 0 {
			t.Error("erroneous value in singleton names for attribute: " + name)
		}
		names[name] = 0
	}

	if len(names) != len(exNames) {
		t.Error("names and exNames len did not match")
	}

	for i := 0; i < len(exNames); i++ {
		names[exNames[i]]++
	}

	for name, count := range names {
		if count != 1 {
			t.Error("mismatch in names and exMatch. Map was missing: " + name)
		}
	}

	// Test basic singleton
	a, ok := c.Map["a"]["a"]["a"]
	if !ok {
		t.Error("can't find 'a' tuple in map")
	}

	if len(a) < 1 || a[0] != "b" {
		t.Error("incorrect value for 'a' in map")
	}

	// Find singleton single tuple
	force, ok := c.Map["force"]["force"]["force"]
	if !ok {
		t.Error("can't find force in map")
	}
	if len(force) > 0 {
		t.Error("erroneous value for 'force' in map")
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

// TestQuoting checks if quoting works as expected
func TestQuoting(t *testing.T) {
	path := testFile
	f, err := os.Open(path)
	if err != nil {
		t.Error("could not open", path, "→", err)
	}

	c, err := Load(f)
	if err != nil {
		t.Error("could not load →", err)
	}
	Quoting = Single

	var first, second strings.Builder
	c.Emit(&first)
	fs := first.String()

	after, err := Load(strings.NewReader(fs))
	if err != nil {
		t.Error("could not load from single-quoted first emission")
	}
	after.Emit(&second)
	ss := second.String()

	if fs != ss {
		t.Error("mismatched single-quoted emissions")
	}

	// Guard?
	Quoting = Double
}
