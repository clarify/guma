package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

var cfg struct {
	output  string
	xmltype string
}

func init() {
	flag.StringVar(&cfg.output, "o", "", "Output to file instead of stdout")
	flag.StringVar(&cfg.xmltype, "t", "", "The type of XML specification")
}

// A single XML file usually contain either a TypeDictionary or a NodeSet at
// the root.
type root struct {
	pname    string
	TypeDict typeDict `xml:"opc:TypeDictionary"`
	NodeSet  nodeSet  `xml:"NodeSet"`
	Services services `xml:"wsdl:definitions"`
}

func (r root) Code() string {
	b := bytes.NewBuffer(nil)
	fmt.Fprint(b, "// Code generated by opcua-xml2code. DO NOT EDIT.\n\n")
	fmt.Fprintf(b, "package %s\n", r.pname)
	fmt.Fprint(b, r.TypeDict.Code())
	fmt.Fprint(b, r.NodeSet.Code())
	fmt.Fprint(b, r.Services.Code())
	return b.String()
}

func main() {
	flag.Parse()

	var out io.Writer
	if cfg.output != "" {
		outFile, err := os.Create(cfg.output)
		if err != nil {
			log.Fatal(err.Error())
		}
		defer outFile.Close()
		out = outFile
	} else {
		out = os.Stdout
	}

	if flag.NArg() < 1 {
		log.Print("Usage:\n  opcua-xml2code FILE\n")
		os.Exit(2)
	}

	var dec *xml.Decoder
	if f, err := os.Open(flag.Arg(0)); err == nil {
		defer f.Close()
		dec = xml.NewDecoder(f)
	} else {
		log.Fatalln(err)
	}

	root := root{}
	var v interface{}
	switch cfg.xmltype {
	case "types":
		root.pname = "uatype"
		v = &root.TypeDict
	case "services":
		root.pname = "stack"
		v = &root.Services
	case "nodes":
		root.pname = "uatype"
		v = &root.NodeSet
	default:
		log.Fatalln("unknown xmltype:", cfg.xmltype)
	}

	if err := dec.Decode(v); err != nil {
		log.Fatalln(err)
	}

	fmt.Fprintln(out, root.Code())
}
