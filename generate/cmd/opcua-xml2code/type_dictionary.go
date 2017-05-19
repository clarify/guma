package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"text/template"
)

var goTypes = map[string]string{
	"Boolean":    "bool",
	"SByte":      "int8",
	"Byte":       "uint8",
	"Int16":      "int16",
	"UInt16":     "uint16",
	"Int32":      "int32",
	"UInt32":     "uint32",
	"Int64":      "int64",
	"UInt64":     "uint64",
	"Float":      "float32",
	"Double":     "float64",
	"CharArray":  "string", // prefixed with length.
	"String":     "String", // zero-terminated.
	"Char":       "rune",   // 32-bit UTF encoded.
	"DateTime":   "time.Time",
	"StatusCode": "enumStatusCode",
}

func goTypeName(s string) string {
	// Trim away "<prefix>:" before lookup.
	if i := strings.IndexRune(s, ':'); i > 0 {
		s = s[1+i:]
	}

	if gs := goTypes[s]; gs != "" {
		return gs
	}
	return s

}

func setGoTypeName(opcName, goName string) {
	goTypes[opcName] = goName
}

func goTypeNameFromBitSize(n int) string {
	switch (n - 1) / 8 {
	case 0: // 1 byte
		return "byte"
	case 1: // 2 bytes
		return "uint16"
	case 3: // 4 bytes
		return "uint32"
	case 7: // 8 bytes
		return "uint64"
	}
	panic(fmt.Sprint("Unexpected bit length:", n))
}

type typeDict struct {
	Enums   []enumType   `xml:"EnumeratedType"`
	Structs []structType `xml:"StructuredType"`
}

func (d typeDict) Code() string {
	b := bytes.NewBuffer(nil)

	for _, e := range d.Enums {
		setGoTypeName(e.Name, e.GoName())
		fmt.Fprintln(b, e.Code())
	}

	for _, s := range d.Structs {
		fmt.Fprintln(b, s.Code())
	}
	return b.String()
}

var enumTmpl = template.Must(template.New("enum.tmpl").Parse(`
type {{.GoName}} {{.GoType}}

{{if .NeedsBitLengther}}// BitLength returns the number of bits that should be used for encoding this value.
func (t {{.GoName}}) BitLength() int {
	return {{.BitLength}}
}
{{end}}
{{$prefix := .Name}}{{$type := .GoName}}{{if .Doc}}// {{.Doc}}
{{end}}const ({{range .Values}}
	{{$prefix}}{{.Name}} {{$type}} = {{.Value}}{{end}}
)`))

type enumType struct {
	Name      string      `xml:"Name,attr"`
	Doc       string      `xml:"Documentation"`
	BitLength int         `xml:"LengthInBits,attr,omitempty"`
	Values    []enumValue `xml:"EnumeratedValue"`
}

func (e enumType) Code() string {
	b := bytes.NewBuffer(nil)
	if err := enumTmpl.Execute(b, e); err != nil {
		log.Println("[ERROR]", err)
	}
	return b.String()
}

// NeedsBitLengther returns true if the value needs to be masked or bitshifted when encoding.
func (e enumType) NeedsBitLengther() bool {
	switch e.BitLength {
	case 0, 8, 16, 32, 64:
		return false
	}
	return true
}

func (e enumType) GoName() string {
	return "enum" + e.Name
}

func (e enumType) GoType() string {
	return goTypeNameFromBitSize(e.BitLength)
}

type enumValue struct {
	Name  string `xml:"Name,attr"`
	Value int    `xml:"Value,attr"`
}

var structTmpl = template.Must(template.New("struct.tmpl").Parse(`
{{if .Doc}}// {{.GoDoc}}
{{end}}type {{.Name}} struct {
	{{if .BaseType}}{{.GoBaseType}}
	{{end}}{{range .Fields}}
	{{.Code}}{{end}}
}`))

type structType struct {
	Name     string        `xml:"Name,attr"`
	BaseType string        `xml:"BaseType,attr"`
	Doc      string        `xml:"Documentation"`
	Fields   []structField `xml:"Field"`
}

func (s structType) Code() string {
	b := bytes.NewBuffer(nil)
	if err := structTmpl.Execute(b, s); err != nil {
		log.Println("[ERROR]", err)
	}
	return b.String()
}

func (s structType) GoBaseType() string {
	return goTypeName(s.BaseType)
}

func (s structType) GoDoc() string {
	return goDoc(s.Name, s.Doc)
}

type structField struct {
	Name     string `xml:"Name,attr"`
	TypeName string `xml:"TypeName,attr"`
	Length   int    `xml:"Length,attr,omitempty"`

	// Fields used for generating struct tags:
	LengthField string `xml:"LengthField,attr,omitempty"`
	SwitchField string `xml:"SwitchField,attr,omitempty"`
	SwitchValue string `xml:"SwitchValue,attr,omitempty"`
}

func (f structField) Code() string {
	return fmt.Sprint(f.Name, " ", f.GoType(), " ", f.GoTags())
}

func (f structField) GoType() string {
	t := goTypeName(f.TypeName)

	if t == "Bit" && f.Length > 1 {
		// In case of Bit array, use appropriate go type and we will add a struct tag in GoTags.
		return goTypeNameFromBitSize(f.Length)
	}

	if f.Length > 1 {
		// Fixed length entity.
		return fmt.Sprintf("[%d]%s", f.Length, t)
	} else if f.LengthField != "" {
		// Variable length array.
		return fmt.Sprintf("[]%s", t)
	} else if t == "DiagnosticInfo" {
		// Hardcoded to avoid infinite recursion.
		return "*" + t
	}
	return t
}

func (f structField) GoTags() string {
	tags := []string{}

	if goTypeName(f.TypeName) == "Bit" && f.Length > 1 {
		tags = append(tags, fmt.Sprintf("bits=%d", f.Length))
	}
	if len(f.LengthField) > 0 {
		tags = append(tags, fmt.Sprintf("lengthField=%s", f.LengthField))
	}
	if len(f.SwitchField) > 0 {
		tags = append(tags, fmt.Sprintf("switchField=%s", f.SwitchField))
	}
	if len(f.SwitchValue) > 0 {
		tags = append(tags, fmt.Sprintf("switchValue=%s", f.SwitchValue))
	}

	if len(tags) > 0 {
		return fmt.Sprintf("`opcua:\"%s\"`", strings.Join(tags, ","))
	}
	return ""
}
