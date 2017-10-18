//go:generate csv2code -o error_auto.go  -csv ../../schemas/1.03/Opc.Ua.StatusCodes.csv errors_auto.go.tmpl
//go:generate gofmt -s -w error_auto.go

package uatype

type enumStatusCode uint32
