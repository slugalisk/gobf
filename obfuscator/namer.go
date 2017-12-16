package obfuscator

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"regexp"
)

// errors
var (
	ErrMultipleAlias = errors.New("received multiple values with existing aliases")
)

var (
	unsafeChars = regexp.MustCompile("[^a-zA-Z]")
)

// Namer ...
type Namer struct {
	length int
	names  map[string]string
	used   map[string]struct{}
}

// NewNamer ...
func NewNamer(length int) *Namer {
	return &Namer{
		length: length,
		names:  make(map[string]string),
		used:   make(map[string]struct{}),
	}
}

func (n *Namer) makeUnique() (string, error) {
	data := make([]byte, n.length*2)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	alias := base64.URLEncoding.EncodeToString(data)
	alias = unsafeChars.ReplaceAllString(alias, "")

	return alias[0:n.length], nil
}

// Assign values to aliases manually
func (n *Namer) Assign(alias string, names ...string) {
	n.used[alias] = struct{}{}
	for _, name := range names {
		n.names[name] = alias
	}
}

// AliasAll assign the same alias to all the names given. If we've already
// aliased one of the names use the existing alias...
func (n *Namer) AliasAll(names []string) (string, error) {
	// avoid aliasing aliases... it's possible, however unlikely, for this to
	// fail if we generate an alias that collides with a name
	var aliased []string
	for i := len(names) - 1; i >= 0; i-- {
		if _, ok := n.used[names[i]]; ok {
			aliased = append(aliased, names[i])

			copy(names[i:], names[i+1:])
			names = names[:len(names)-1]
		}
	}

	if len(aliased) > 1 {
		return "", ErrMultipleAlias
	}

	if len(aliased) == 1 {
		n.Assign(aliased[0], names...)
		return aliased[0], nil
	}

	// copy existing alias
	for _, name := range names {
		if alias, ok := n.names[name]; ok {
			n.Assign(alias, names...)
			return alias, nil
		}
	}

	// generate new alias
	for {
		alias, err := n.makeUnique()
		if err != nil {
			return "", err
		}

		if _, ok := n.used[alias]; !ok {
			n.used[alias] = struct{}{}
			n.Assign(alias, names...)
			return alias, nil
		}
	}
}

// Alias ...
func (n *Namer) Alias(name string) (string, error) {
	return n.AliasAll([]string{name})
}
