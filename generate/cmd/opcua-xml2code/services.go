package main

import (
	"bytes"
	"fmt"
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
func (c *Client) {{.GoName}}(req uatype.{{.GoReq}}) (*uatype.{{.GoResp}}, error) {
	var buf bytes.Buffer
	enc := binary.NewEncoder(&buf)
	if err := enc.Encode(req); err != nil {
		return nil, err
	}

	resp, err := c.Channel.Send(&transport.Request{
		NodeID: {{.NodeID}},
		Body: &buf,
	})
	if err != nil {
		return nil, err
	}

	res := &uatype.{{.GoResp}}{}
	if err := binary.NewDecoder(resp.Body).Decode(res); err != nil {
		return nil, err
	}

	return res, nil
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
	return strings.Replace(o.Input.Name, "Message", "Request", 1)
}

func (o operation) GoResp() string {
	return strings.Replace(o.Input.Name, "Message", "Response", 1)
}

func (o operation) NodeID() string {
	return fmt.Sprintf("uatype.NewFourByteNodeID(0, uatype.NodeId%sRequest_Encoding_DefaultBinary)", o.Name)
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
