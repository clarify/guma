package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/template"
)

var nodeSetTmpl = template.Must(template.New("node_set.tmpl").Parse(`

func init() {
	{{range .Nodes}}nodeInfoMap[{{.GoValue}}] = nodeInfo{
		displayName: "{{.DisplayName}}",
		class: {{.GoClass}},
		description: "{{.Doc}}",
	}
{{end}}}
`))

type nodeSet struct {
	Nodes []node `xml:"Nodes>Node"`
}

func (ns nodeSet) Code() string {
	if len(ns.Nodes) == 0 {
		return ""
	}
	b := bytes.NewBuffer(nil)
	if err := nodeSetTmpl.Execute(b, ns); err != nil {
		log.Println("[ERROR]", err)
	}

	return b.String()
}

// node includes a very limited set of information that we are interested in
// reading from an XML Node entry when generating code.
type node struct {
	Identifier  string `xml:"NodeId>Identifier"`
	DisplayName string `xml:"DisplayName>Text"`
	Name        string `xml:"BrowseName>Name"`
	Class       string `xml:"NodeClass"`
	Doc         string `xml:"Description>Text"`
}

func (n node) GoValue() string {
	if !strings.HasPrefix(n.Identifier, "i=") {
		panic(fmt.Sprintf("Unsupported identifier prefix for '%s'", n.Identifier))
	}
	return n.Identifier[2:]
}

func (n node) GoClass() string {
	i := strings.IndexRune(n.Class, '_')
	return "NodeClass" + n.Class[:i]
}
