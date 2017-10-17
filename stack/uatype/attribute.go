//go:generate csv2code -o attribute_auto.go -csv ../schemas/1.03/AttributeIds.csv attribute_auto.go.tmpl
//go:generate gofmt -s -w attribute_auto.go

package uatype
