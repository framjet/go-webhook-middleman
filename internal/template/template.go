package template

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"github.com/framjet/go-webhook-middleman/internal/config"
	"net/http"
	"text/template"
)

type TemplateContext struct {
	Params    map[string]string
	Variables map[string]string
	Body      string
	Route     config.Route
	Request   http.Request
}

func RenderTemplate(tmpl string, ctx TemplateContext) (string, error) {
	if tmpl == "" {
		return "", nil
	}

	t, err := template.New("webhook").Funcs(sprig.FuncMap()).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}

	var buf bytes.Buffer
	data := map[string]interface{}{
		"params":  ctx.Params,
		"var":     ctx.Variables,
		"body":    ctx.Body,
		"route":   ctx.Route,
		"request": ctx.Request,
	}

	err = t.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("template execute error: %w", err)
	}

	return buf.String(), nil
}
