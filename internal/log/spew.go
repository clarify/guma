package log

import "github.com/davecgh/go-spew/spew"

// SpewFmt default implementation.
var SpewFmt = spew.ConfigState{
	Indent:   "\t",
	SortKeys: true,
}
