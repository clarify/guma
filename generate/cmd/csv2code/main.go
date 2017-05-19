package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
)

type csvRow []string

var cfg struct {
	csv     string
	sep     string
	groupBy int
	output  string
}

func init() {
	log.SetOutput(os.Stderr)

	flag.StringVar(&cfg.csv, "csv", "", "Path to CSV file")
	flag.StringVar(&cfg.sep, "sep", ",", "Column separator")
	flag.IntVar(&cfg.groupBy, "group-by", -1, "Group data by a column zero-based index")
	flag.StringVar(&cfg.output, "o", "", "Output to file instead of stdout")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "Usage csv2code:\n  csv2code [flags] templates, ...\n\n")
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()

	}
}

func main() {
	var out io.Writer

	// Parse input flags.
	flag.Parse()
	tmplSrc := flag.Args()
	if len(tmplSrc) < 1 {
		flag.Usage()
		os.Exit(2)
	}
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

	// Read input into rows.
	rows := make([]csvRow, 0, 1024)
	if csvSrc, err := os.Open(cfg.csv); err == nil {
		scanner := bufio.NewScanner(csvSrc)
		for scanner.Scan() {
			bits := strings.Split(scanner.Text(), cfg.sep)
			rows = append(rows, bits)
		}
	} else {
		log.Fatal("can't read CSV:", err)
	}

	// Compose data structure for template.
	var data interface{}
	if gi := cfg.groupBy; gi > -1 {
		rowMap := make(map[string][]csvRow)
		for i, row := range rows {
			if len(row) < gi {
				log.Printf("skipping CSV row %d: to few columns", i)
				continue
			}
			rowMap[row[gi]] = append(rowMap[row[gi]], row)
			data = rowMap
		}
	} else {
		data = rows
	}

	// Parse template file(s).
	tmpl := template.New("").Funcs(template.FuncMap{
		"atoi": strconv.Atoi,
		"itoa": strconv.Itoa,
	})
	for _, filename := range tmplSrc {
		var err error
		var b []byte
		var s string
		b, err = ioutil.ReadFile(filename)
		if err != nil {
			log.Fatalf("can't read file '%s': %s", filename, err)
		}

		s = string(b)
		if _, err = tmpl.Parse(s); err != nil {
			log.Fatalf("can't parse template '%s': %s", filename, err)
		}
	}

	// Render templates.
	fmt.Fprint(out, "// DO NOT EDIT. Generated by csv2code.\n\n")
	if err := tmpl.Execute(out, data); err != nil {
		log.Fatal("failed to execute template:", err)
	}
}
