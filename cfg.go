// Copyright (c) 2021, Microsoft Corporation, Sean Hinchee
// Licensed under the MIT License.

// Package cfg is an implementation of cfg(2) in Go: http://man.postnix.pw/purgatorio/2/cfg.
package cfg

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"unicode"
)

const (
	commitSize = 100 // Number of attributes to buffer.
)

var (
	// Chatty controls verbose parser output.
	Chatty = false
)

// States that the parser  can be in at a given time.
type states int

const (
	name        states = iota // name=
	value                     // name=val
	equals                    // =
	squotebegin               // In a 'foo'
	dquotebegin               // In a "bar"
	squoteend                 // Closed a 'foo'
	dquoteend                 // Closed a "bar"
)

// Cfg is a data structure representation of a cfg(2) file.
type Cfg struct {
	Records []*Record
}

// Attribute is a name and optional value pair.
type Attribute struct {
	Name  string // Mandatory
	Value string // Optional
}

// Tuple represents a set of attributes which contain names and optional value pairs.
type Tuple struct {
	Attributes []*Attribute
}

// Record represents a set of tuples which contain attributes.
type Record struct {
	Tuples []*Tuple
}

// Lookup returns the attributes whose name matches 'name'.
func (t *Tuple) Lookup(name string) ([]*Attribute, bool) {
	var out []*Attribute

	for _, a := range t.Attributes {
		if a.Name == name {
			out = append(out, a)
		}
	}

	return out, len(out) > 0
}

// PrimaryKey returns the first name of the first attribute of a tuple.
func (t Tuple) PrimaryKey() string {
	return t.Attributes[0].Name
}

// Map returns a map[string]string representation of a tuple.
// Only the first instance of a name is inserted.
func (t Tuple) Map() map[string]string {
	out := make(map[string]string)
	for _, a := range t.Attributes {
		if _, ok := out[a.Name]; ok {
			// Entry exists
			continue
		}

		out[a.Name] = a.Value
	}
	return out
}

// Lookup returns cfg tuples whose primary key matches 'name'.
func (r *Record) Lookup(name string) ([]*Tuple, bool) {
	var out []*Tuple

	for _, t := range r.Tuples {
		if t.PrimaryKey() == name {
			out = append(out, t)
		}
	}

	return out, len(out) > 0
}

// PrimaryKey returns the first name of the first attribute of the first tuple of a record.
func (r Record) PrimaryKey() string {
	return r.Tuples[0].PrimaryKey()
}

// Maps returns the set of map representations of its tuples.
func (r Record) Maps() []map[string]string {
	var out []map[string]string
	for _, t := range r.Tuples {
		out = append(out, t.Map())
	}
	return out
}

// FlatMap returns a map which is the union of all the record's tuples' maps.
// Only the first instance of a name is inserted.
func (r Record) FlatMap() map[string]string {
	out := make(map[string]string)
	for _, t := range r.Tuples {
		for n, v := range t.Map() {
			if _, ok := out[n]; ok {
				// This entry exists
				continue
			}

			out[n] = v
		}
	}
	return out
}

// Lookup returns cfg records whose primary key matches 'name'.
func (c *Cfg) Lookup(name string) ([]*Record, bool) {
	var out []*Record

	for _, r := range c.Records {
		if r.PrimaryKey() == name {
			out = append(out, r)
		}
	}

	return out, len(out) > 0
}

// Keys returns the Record primary keys for a cfg.
func (c *Cfg) Keys() []string {
	var out []string

	for _, r := range c.Records {
		out = append(out, r.PrimaryKey())
	}

	return out
}

// FlatMap returns a map which is the union of all the cfg's records' tuples' maps.
// Only the first instance of a name is inserted.
func (c Cfg) FlatMap() map[string]string {
	out := make(map[string]string)
	for _, r := range c.Records {
		for _, t := range r.Tuples {
			for n, v := range t.Map() {
				if _, ok := out[n]; ok {
					// This entry exists
					continue
				}

				out[n] = v
			}
		}
	}
	return out
}

// Load parses a cfg file and returns a complete cfg.
func Load(r io.Reader) (Cfg, error) {
	c := Cfg{}
	br := bufio.NewReader(r)
	var ln, rn uint64

lines:
	for ln = 1; ; ln++ {
		line, err := br.ReadString('\n')
		if err == io.EOF {
			break lines
		}
		if err != nil {
			return c, err
		}

		// Trim comments
		ci := strings.IndexFunc(line, func(r rune) bool {
			return r == '#'
		})
		if ci >= 0 {
			line = line[:ci]
		}

		// Whitespace beginning index and first 'letter' index
		wi := strings.IndexFunc(line, unicode.IsSpace)
		li := strings.IndexFunc(line, func(r rune) bool {
			return !unicode.IsSpace(r)
		})

		in := false

		if wi < li {
			// Leading whitespace, Tuple is a part of a record
			chat("tuple in record →", line)
			in = true

		} else if (wi < 0 || wi > li) && li >= 0 {
			// No leading whitespace, start a new record
			chat("new record →", line)
			in = false

		} else {
			// Empty line
			chat("empty →", line)
			continue lines
		}

		done := make(chan *Tuple)
		commit := make(chan *Attribute, commitSize)
		go func() {
			tuple := &Tuple{[]*Attribute{}}
			for {
				a, ok := <-commit
				if !ok {
					break
				}

				// Discard empty attributes (usually a bug)
				if a.Name == "" && a.Value == "" {
					continue
				}

				// Insert attribute
				tuple.Attributes = append(tuple.Attributes, a)
			}
			done <- tuple
		}()

		// Parse line
		state := name
		lr := strings.NewReader(line)

		n := ""
		v := ""
		var word strings.Builder
	scan:
		for rn = 1; lr.Len() > 0; rn++ {
			r, _, err := lr.ReadRune()
			chat(fmt.Sprintf("%c ⇒ %v\n", r, state))
			if err == io.EOF {
				switch state {
				case value:
					// Finish the value
					v = word.String()
					word.Reset()
					commit <- &Attribute{n, v}
					n = ""
					v = ""

				default:
					break scan
				}
			}
			if err != nil {
				return c, err
			}

			switch {
			case unicode.IsSpace(r):
				switch state {
				case squotebegin:
					fallthrough
				case dquotebegin:
					word.WriteRune(r)

				case squoteend:
					fallthrough
				case dquoteend:
					fallthrough
				case value:
					// Finish a value
					v = word.String()
					word.Reset()
					commit <- &Attribute{n, v}
					n = ""
					v = ""
					state = name

				case equals:
					// A name without a value was had, now this is a new name
					word.Reset()
					commit <- &Attribute{n, v}
					n = ""
					v = ""
					state = name

				case name:
					// A space after a name, for optional '=' after valueless name
					// Finish a name
					n = word.String()
					word.Reset()
					commit <- &Attribute{n, v}
					n = ""
					v = ""
					state = name

				default:
				}
				continue scan

			case r == '=':
				switch state {
				// When in quotes, append
				case squotebegin:
					fallthrough
				case dquotebegin:
					word.WriteRune('=')

				case name:
					// Finish the name, no spaces here
					n = word.String()
					word.Reset()

					state = equals

				default:
					state = equals
					continue scan
				}

			case r == '\'':
				next, _, err := lr.ReadRune()
				if err == io.EOF {
					return c, errors.New("unclosed single quote (') at EOF")
				}
				if err != nil {
					return c, err
				}

				literal := false
				if next == '\'' {
					literal = true
					rn++
				} else {
					lr.UnreadRune()
				}

				if literal || state == dquotebegin {
					// We are inserting a literal single quote
					// 'foo '' bar' ⇒ foo ' bar
					word.WriteRune('\'')
					continue scan
				}

				switch state {
				case squotebegin:
					// Commit the word
					if n == "" {
						// We are the name
						n = word.String()
						word.Reset()

					} else {
						// We are the value
						v = word.String()
						word.Reset()
						commit <- &Attribute{n, v}
						n = ""
						v = ""
					}
					state = squoteend

				case name:
					// Guard if word is empty
					if word.Len() < 1 {
						state = squotebegin
						continue scan
					}

					// A name preceded us, commit it
					n = word.String()
					word.Reset()
					commit <- &Attribute{n, v}
					n = ""
					v = ""
					state = squotebegin

				default:
					state = squotebegin
				}

			case r == '"':
				next, _, err := lr.ReadRune()
				if err == io.EOF {
					return c, errors.New("unclosed double quote (\") at EOF")
				}
				if err != nil {
					return c, err
				}

				literal := false
				if next == '"' {
					literal = true
					rn++
				} else {
					lr.UnreadRune()
				}

				if literal || state == squotebegin {
					// We are inserting a literal double quote
					// "foo "" bar" ⇒ foo " bar
					word.WriteRune('"')
					continue scan
				}

				switch state {
				case dquotebegin:
					// Commit the word
					if n == "" {
						// We are the name
						n = word.String()
						word.Reset()

					} else {
						// We are the value
						v = word.String()
						word.Reset()
						commit <- &Attribute{n, v}
						n = ""
						v = ""
					}
					state = dquoteend

				case name:
					// Guard if word is empty
					if word.Len() < 1 {
						state = dquotebegin
						continue scan
					}

					// A name preceded us, commit it
					n = word.String()
					word.Reset()
					commit <- &Attribute{n, v}
					n = ""
					v = ""
					state = dquotebegin

				default:
					state = dquotebegin
				}

			default:
				// Part of a name or value
				switch state {
				case equals:
					state = value
				}
				word.WriteRune(r)
			}
		}
		close(commit)
		tuple := <-done

		pos := fmt.Sprintf("near line:rune of %d:%d", ln, rn)
		switch state {
		case squotebegin:
			return c, errors.New(`unterminated single quote (') ` + pos)
		case dquotebegin:
			return c, errors.New(`unterminated double quote (") ` + pos)
		}

		// Tuple is finished
		if in {
			// Append Tuple to last record
			last := len(c.Records) - 1
			if last < 0 {
				return c, errors.New("no parent record for indented tuple, the first tuple must be unindented and thus start a record " + pos)
			}

			c.Records[last].Tuples = append(c.Records[last].Tuples, tuple)

		} else {
			// New Record with just this tuple
			c.Records = append(c.Records, &Record{[]*Tuple{tuple}})
		}
	}

	return c, nil
}

// Emit takes writes the Cfg's string representation to 'w'.
func (c Cfg) Emit(w io.Writer) {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	bw.WriteString(c.String())
}

/* Stringification routines */

func (c Cfg) String() (out string) {
	for _, r := range c.Records {
		out += r.String()
	}

	return
}

func (r Record) String() (out string) {
	out += r.Tuples[0].String() + "\n"

	if len(r.Tuples) > 1 {
		for _, t := range r.Tuples[1:] {
			out += "	" + t.String() + "\n"
		}
	}

	return
}

func (t Tuple) String() (out string) {
	for _, a := range t.Attributes {
		out += a.String() + " "
	}
	return
}

func (a Attribute) String() (out string) {
	nf := strings.Fields(a.Name)
	if len(nf) > 1 {
		// Quote it
		out += `"` + strings.ReplaceAll(a.Name, `"`, `""`) + `"`
	} else {
		out += a.Name
	}

	out += "="

	vf := strings.Fields(a.Value)
	if len(vf) > 1 {
		// Quote it
		out += `"` + strings.ReplaceAll(a.Value, `"`, `""`) + `"`
	} else {
		out += a.Value
	}

	return
}

func (s states) String() string {
	switch s {
	case name:
		return "Name"
	case value:
		return "Value"
	case squotebegin:
		return "'Begin"
	case squoteend:
		return "'End"
	case dquotebegin:
		return `"Begin`
	case dquoteend:
		return `"End`
	case equals:
		return "Equals"
	default:
		return "UNKNOWN"
	}
}

// Verbose logging for parser debugging
func chat(s ...interface{}) {
	if !Chatty {
		return
	}

	log.Println(s...)
}
