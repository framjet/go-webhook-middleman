package templateRenderer

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/framjet/go-webhook-middleman/internal/config"
	wmSprout "github.com/framjet/go-webhook-middleman/internal/sprout"
	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"
	"github.com/go-sprout/sprout/registry/backward"
	"net/http"
	"text/template"
)

var (
	tplRenderer = NewTemplateRenderer()
)

type TemplateRenderer struct {
	FunctionMap template.FuncMap
}

func NewTemplateRenderer() *TemplateRenderer {
	handler := sprout.New()
	handler.AddGroups(all.RegistryGroup())
	handler.AddRegistry(backward.NewRegistry())
	handler.AddRegistry(wmSprout.NewRegistry())

	return &TemplateRenderer{
		FunctionMap: handler.Build(),
	}
}

type TemplateContext struct {
	Params    map[string]string
	Variables map[string]string
	Body      string
	Route     config.Route
	Request   http.Request
}

func GetTplRenderer() *TemplateRenderer {
	return tplRenderer
}

func RenderTemplate(tmpl string, resolved config.ResolvedDestination, ctx TemplateContext) (string, error) {
	if tmpl == "" {
		return "", nil
	}

	t, err := template.New("webhook").Funcs(GetTplRenderer().FunctionMap).Parse(tmpl)
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
		"resolved": map[string]interface{}{
			"method":  resolved.Method,
			"headers": resolved.Headers,
			"body":    string(resolved.Body),
		},
	}

	err = t.Execute(&buf, data)
	if err != nil {
		if errors.Is(err, wmSprout.GetErrTemplateStopped()) {
			return "", err
		}

		return "", fmt.Errorf("template execute error: %w", err)
	}

	return buf.String(), nil
}
