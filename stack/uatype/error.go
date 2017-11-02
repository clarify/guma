//go:generate csv2code -o error_auto.go  -csv ../../schemas/1.03/Opc.Ua.StatusCodes.csv errors_auto.go.tmpl
//go:generate gofmt -s -w error_auto.go

package uatype

// StatusCode is expected to hold any value defined in the StatusCode<...> const
// block.
type StatusCode uint32

// StatusText returns a text for the OPC UA status code. It returns the empty
// string if the code is unknown.
func StatusText(code StatusCode) string {
	return statusText[code]
}
