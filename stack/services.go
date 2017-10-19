//go:generate opcua-xml2code -o services_auto.go -t=services ../schemas/1.03/Opc.Ua.Services.wsdl
//go:generate goimports -w services_auto.go

package stack
