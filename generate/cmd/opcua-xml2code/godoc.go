package main

import (
	"strings"
)

// goDoc returns a go-style documentation string for the named type.
func goDoc(name, doc string) string {
	if strings.HasPrefix(doc, "A ") {
		return name + " is a " + doc[2:]
	}
	if strings.HasPrefix(doc, "An ") {
		return name + " is an " + doc[3:]
	}
	if strings.HasPrefix(doc, "The ") {
		return name + " the " + doc[4:]
	}
	return name + " " + strings.ToLower(doc[:1]) + doc[1:]
}
