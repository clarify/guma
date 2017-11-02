package main

import (
	"bytes"
	"log"
	"strings"
	"text/template"
)

type services struct {
	Operations []operation `xml:"portType>operation"`
}

var servicesTmlp = template.Must(template.New("services.tmpl").Parse(`
{{range .Operations}}
{{if .Include}}
// {{.GoName}} sends a {{.GoName}} request to the server, and waits for a
// response or timeout. To send and receive with no timeout use the zero time
// as deadline.
func (c *Client) {{.GoName}}(req uatype.{{.GoReq}}, deadline time.Time) (*uatype.{{.GoResp}}, error) {
	var buf bytes.Buffer

	if err := binary.NewEncoder(&buf).Encode(req); err != nil {
		return nil, err
	}

	resp, err := c.Channel.Send(transport.Request{
		NodeID: uatype.NewFourByteNodeID(0, {{.RequestNodeID}}).Expanded(),
		Body: &buf,
	}, deadline)
	if err != nil {
		return nil, err
	}
	switch resp.NodeID.Uint() {
	case {{.ResponseNodeID}}:
		res := &uatype.{{.GoResp}}{}
		if err := binary.NewDecoder(resp.Body).Decode(res); err != nil {
			return res, err
		}
		return res, nil
	case {{.FaultNodeID}}:
		fault := uatype.{{.GoFault}}{}
		if err := binary.NewDecoder(resp.Body).Decode(&fault); err != nil {
			return nil, err
		}
		return nil, fault
	}
	return nil, fmt.Errorf("Unexpected NodeID: %d %s", resp.NodeID.Uint(), resp.NodeID.DisplayName())
}{{end}}
{{end}}`))

func (s services) Code() string {
	b := bytes.NewBuffer(nil)
	if err := servicesTmlp.Execute(b, s); err != nil {
		log.Println("[ERROR]", err)
	}
	return b.String()
}

type operation struct {
	Name   string `xml:"name,attr"`
	Input  action `xml:"input"`
	Output action `xml:"output"`
	Fault  action `xml:"fault"`
}

func (o operation) GoName() string {
	return o.Name
}

func (o operation) GoReq() string {
	// This might break for future versions of OPC UA, but works for 1.03.
	// Instead of fetching the propper Request type from the message definition
	// section of the XML, we simply deduce it from the input message name.
	return strings.Replace(o.Input.Name, "Message", "Request", 1)
}

func (o operation) GoResp() string {
	// This might break for future versions of OPC UA, but works for 1.03.
	// Instead of fetching the propper Response type from the message definition
	// section of the XML, we simply deduce it from the output message name.
	return strings.Replace(o.Output.Name, "Message", "", 1)
}

func (o operation) GoFault() string {
	// This might break for future versions of OPC UA, but works for 1.03,
	// where all fault message definitions uses the ServiceFault response type.
	return "ServiceFault"
}

func (o operation) RequestNodeID() string {
	return "uatype.NodeId" + o.Name + "Request_Encoding_DefaultBinary"
}

func (o operation) ResponseNodeID() string {
	return "uatype.NodeId" + o.Name + "Response_Encoding_DefaultBinary"
}

func (o operation) FaultNodeID() string {
	return "uatype.NodeIdServiceFault_Encoding_DefaultBinary"
}

func (o operation) Include() bool {
	if o.Name == "InvokeService" {
		return false
	}

	return true
}

type action struct {
	Name    string `xml:"name,attr"`
	Message string `xml:"message,attr"`
}
