# cfg

[![PkgGoDev](https://pkg.go.dev/badge/github.com/seh-msft/cfg)](https://pkg.go.dev/github.com/seh-msft/cfg)

Package "cfg" provides functionality for the parsing and emitting of [cfg(2)](http://man.postnix.pw/purgatorio/2/cfg) files. 

Written in [Go](https://golang.org/). 

## Test

	go test

## Install

	go install

## File structuring

Cfg files are structured in sets of lines referred to as records of tuples. 

A tuple is a line which consists of one or more `name=value` form attributes. 

There must be no spaces around the `=` for a `name=value` pair. 

A primary key is the first name in the first tuple in a record. 

Values are optional. Optional values may be useful for indicating an explicit primary key. 

If a value will be omitted, the `=` after the name is optional. 

That is, all of the following are valid:

```
'foo'
	"bar" first 'second third'
test=
	where=
	"when"
```

The following is four names and one value:

```
a= 'b c' d= last=bit
```

Tuples whose first character has whitespace preceding it (are indented) are considered part of the last extant record. The first tuple in a file must not be indented. 

Lookup calls are used by navigating primary keys, but return full tuples. 

The boolean return value for Lookup methods is an 'ok' value indicating if anything was found. 

Names or values may be quoted with single (`'`) or double (`"`) quotes. 

A quote pair inserted a literal quote of that type, that is:

	'alice''s comment'

Becomes:

	alice's comment

Comments (`#`) and empty lines are ignored. 

## Examples

For an example cfg file, see [test.cfg](./test.cfg) and [users.cfg](./users.cfg). 

Condensed, this file would have 2 records, 5 tuples, and 6 attributes:

```
"my network"
	ip=1.2.3.4

creds
	user=alice
	method=key	file="./my_key.pem" 
```

Load from a cfg file:

```go
path := "./test.cfg"
f, err := os.Open(path)
if err != nil {
	log.Fatal("could not open", path, "→", err)
}

c, err := Load(f)
if err != nil {
	log.Fatal("could not load cfg →", err)
}

fmt.Print("We read in:\n", cfg)
```

Get a flattened map of attributes in a record, discarding duplicates:

```go
path := "./test.cfg"
f, err := os.Open(path)
if err != nil {
	log.Fatal("could not open", path, "→", err)
}

c, err := Load(f)
if err != nil {
	log.Fatal("could not load →", err)
}

creds, ok := c.Lookup("creds")
if !ok {
	log.Fatal("No records keyed as 'creds' not found")
}

// Safe since 'ok' guarded us
attrs := creds[0].FlatMap()
for name, value := range attrs {
	fmt.Println(name, "⇒", value)
}
```

## Usage

Generated with `go doc -all`. 

```
package cfg // import "."

Package cfg is an implementation of cfg(2) in Go:
http://man.postnix.pw/purgatorio/2/cfg.

VARIABLES

var (
	// Chatty controls verbose parser output.
	Chatty = false
)

TYPES

type Attribute struct {
	Name  string // Mandatory
	Value string // Optional
}
    Attribute is a name and optional value pair

func (a Attribute) String() (out string)

type Cfg struct {
	Records []*Record
}
    Cfg is a data structure representation of a cfg(2) file.

func Load(r io.Reader) (Cfg, error)
    Load parses a cfg file and returns a complete cfg.

func (c Cfg) Emit(w io.Writer)
    Emit takes writes the Cfg's string representation to 'w'.

func (c Cfg) FlatMap() map[string]string
    FlatMap returns a map which is the union of all the cfg's records' tuples'
    maps. Only the first instance of a name is inserted.

func (c *Cfg) Keys() []string
    Keys returns the Record primary keys for a cfg.

func (c *Cfg) Lookup(name string) ([]*Record, bool)
    Lookup returns cfg records whose primary key matches 'name'.

func (c Cfg) String() (out string)

type Record struct {
	Tuples []*Tuple
}
    Record represents a set of tuples which contain attributes.

func (r Record) FlatMap() map[string]string
    FlatMap returns a map which is the union of all the record's tuples' maps.
    Only the first instance of a name is inserted.

func (r *Record) Lookup(name string) ([]*Tuple, bool)
    Lookup returns cfg tuples whose primary key matches 'name'.

func (r Record) Maps() []map[string]string
    Maps returns the set of map representations of its tuples.

func (r Record) PrimaryKey() string
    PrimaryKey returns the first name of the first attribute of the first tuple
    of a record.

func (r Record) String() (out string)

type Tuple struct {
	Attributes []*Attribute
}
    Tuple represents a set of attributes which contain names and optional value
    pairs.

func (t *Tuple) Lookup(name string) ([]*Attribute, bool)
    Lookup returns the attributes whose name matches 'name'.

func (t Tuple) Map() map[string]string
    Map returns a map[string]string representation of a tuple. Only the first
    instance of a name is inserted.

func (t Tuple) PrimaryKey() string
    PrimaryKey returns the first name of the first attribute of a tuple.

func (t Tuple) String() (out string)
```