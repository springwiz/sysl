package importer

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

type OutputData struct {
	AppName string
	Package string
}

type SyslInfo struct {
	OutputData

	Title       string
	Description string
	OtherFields []string // Ordered key, val pair
}

type MethodEndpoints struct {
	Method    string // one of GET, POST, PUT, OPTION, etc
	Endpoints []Endpoint
}

type writer struct {
	io.Writer
	ind    *IndentWriter
	logger *logrus.Logger
}

func newWriter(out io.Writer, logger *logrus.Logger) *writer {
	return &writer{
		Writer: out,
		ind:    NewIndentWriter("    ", out),
		logger: logger,
	}
}

func (w *writer) Write(info SyslInfo, types TypeList, endpointBasePath string, endpoints ...MethodEndpoints) error {

	if err := w.writeHeader(info); err != nil {
		return err
	}

	if endpointBasePath != "" {
		w.writeLines(fmt.Sprintf("%s:", endpointBasePath), PushIndent, BlankLine)
	}
	for _, method := range endpoints {
		for _, ep := range method.Endpoints {
			w.writeEndpoint(method.Method, ep)
			w.writeLines(BlankLine)
		}
	}
	if endpointBasePath != "" {
		w.writeLines(PopIndent)
	}

	w.writeDefinitions(types)

	return nil
}

func (w *writer) writeHeader(info SyslInfo) error {

	w.mustWrite(`##########################################
##                                      ##
##  AUTOGENERATED CODE -- DO NOT EDIT!  ##
##                                      ##
##########################################

`)
	title := info.Title

	w.writeLines(fmt.Sprintf("%s %s [package=%s]:", info.AppName, quote(title), quote(info.Package)))
	w.ind.Push()

	for i := 0; i < len(info.OtherFields); i += 2 {
		key := info.OtherFields[i]
		val := info.OtherFields[i+1]
		if val != "" {
			w.writeLines(fmt.Sprintf("@%s = %s", key, quote(val)))
		}
	}
	w.writeLines("@description =:", PushIndent, "| "+getDescription(info.Description), PopIndent, BlankLine)

	return nil
}

func buildQueryString(params []Param) string {
	query := ""
	if len(params) > 0 {
		var parts []string
		for _, p := range params {
			optional := ""
			if p.Optional {
				optional = "?"
			}
			parts = append(parts, fmt.Sprintf("%s=%s%s", p.Name, p.Type.Name(), optional))
		}
		query = " ?" + strings.Join(parts, "&")
	}
	return query
}

func buildRequestBodyString(params []Param) string {
	body := ""
	if len(params) > 0 {
		sort.SliceStable(params, func(i, j int) bool {
			return strings.Compare(params[i].Name, params[j].Name) < 0
		})
		var parts []string
		for _, p := range params {
			parts = append(parts, fmt.Sprintf("%s <: %s [~body]", p.Name, p.Type.Name()))
		}
		body = strings.Join(parts, ", ")
	}
	return body
}

func buildRequestHeadersString(params []Param) string {
	headers := ""
	if len(params) > 0 {
		var parts []string
		for _, p := range params {
			optional := map[bool]string{true: "~optional", false: "~required"}[p.Optional]

			safeName := strings.ToLower(strings.ReplaceAll(p.Name, "-", "_"))
			text := fmt.Sprintf("%s <: %s [~header, %s, name=%s]", safeName, p.Type.Name(), optional, quote(p.Name))
			parts = append(parts, text)
		}
		headers = strings.Join(parts, ", ")
	}
	return headers
}

func buildPathString(path string, params []Param) string {

	result := path

	for _, p := range params {
		replacement := fmt.Sprintf("{%s<:%s}", p.Name, p.Type.Name())
		result = strings.ReplaceAll(result, fmt.Sprintf("{%s}", p.Name), replacement)
	}

	return result
}

func (w *writer) writeEndpoint(method string, endpoint Endpoint) {

	header := buildRequestHeadersString(endpoint.Params.HeaderParams())
	body := buildRequestBodyString(endpoint.Params.BodyParams())
	reqStr := ""
	if len(header) > 0 && len(body) > 0 {
		reqStr = fmt.Sprintf(" (%s)", strings.Join([]string{body, header}, ", "))
	} else if len(header) > 0 || len(body) > 0 {
		reqStr = fmt.Sprintf(" (%s)", body+header)
	}

	pathStr := buildPathString(endpoint.Path, endpoint.Params.PathParams())

	w.writeLines(fmt.Sprintf("%s:", pathStr), PushIndent,
		fmt.Sprintf("%s%s%s:", method, reqStr, buildQueryString(endpoint.Params.QueryParams())), PushIndent,
		fmt.Sprintf("| %s", getDescription(endpoint.Description)))

	if len(endpoint.Responses) > 0 {
		var outs []string
		for _, resp := range endpoint.Responses {
			if resp.Type != nil {
				outs = append(outs, getSyslTypeName(resp.Type))
			} else {
				outs = append(outs, resp.Text)
			}
		}
		sort.Strings(outs)
		w.writeLines(fmt.Sprintf("return %s", strings.Join(outs, ", ")))
	}

	w.writeLines(PopIndent, PopIndent)
}

func (w *writer) writeDefinitions(types TypeList) {

	w.writeLines("#" + strings.Repeat("-", 75))
	w.writeLines("# definitions")
	var others []Type
	for _, t := range types {
		_, isEnum := t.(*Enum)
		switch {
		case isEnum:
			// We want the enum aliases listed with the real types
			w.writeLines(BlankLine)
			w.writeExternalAlias(t)
		case !isExternalAlias(t):
			w.writeLines(BlankLine)
			w.writeDefinition(t.(*StandardType))
		default:
			others = append(others, t)
		}
	}
	for _, t := range others {
		w.writeLines(BlankLine)
		w.writeExternalAlias(t)
	}
}

func (w *writer) writeDefinition(t *StandardType) {
	bangName := "type"
	w.writeLines(fmt.Sprintf("!%s %s:", bangName, getSyslTypeName(t)))
	for _, prop := range t.Properties {
		suffix := ""
		if prop.Optional {
			suffix = "?"
		}

		name := prop.Name
		if IsKeyword(name) {
			name += "_"
		}

		w.writeLines(PushIndent, fmt.Sprintf("%s <: %s%s:", name, getSyslTypeName(prop.Type), suffix))
		w.writeLines(PushIndent, fmt.Sprintf("@json_tag = %s", quote(prop.Name)))
		w.writeLines(PopIndent, PopIndent)
	}
}

func (w *writer) writeExternalAlias(item Type) {
	aliasType := "string"
	aliasName := getSyslTypeName(item)
	switch t := item.(type) {
	case *StandardType:
		if len(t.Properties) > 0 {
			aliasType = getSyslTypeName(t.Properties[0].Type)
		}
	case *Alias:
		aliasType = getSyslTypeName(t.Target)
	case *Array:
		aliasType = getSyslTypeName(item)
		aliasName = t.name
	}
	w.writeLines(fmt.Sprintf("!alias %s:", aliasName),
		PushIndent, aliasType, PopIndent)
}

func (w *writer) mustWrite(s string) {
	if _, err := w.Writer.Write([]byte(s)); err != nil {
		w.logger.Fatalf("failed to complete write: %s", err.Error())
	}
}

const PushIndent = "&& >>"
const PopIndent = "&& <<"
const BlankLine = "&& !!"

func (w *writer) writeLines(lines ...string) {
	for _, l := range lines {
		switch l {
		case PushIndent:
			w.ind.Push()
		case PopIndent:
			w.ind.Pop()
		case BlankLine:
			w.mustWrite("\n")
		default:
			_ = w.ind.Write() // nolint: errcheck
			w.mustWrite(l + "\n")
		}
	}
}
