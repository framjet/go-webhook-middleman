package sprout

import (
	"fmt"
	"github.com/go-sprout/sprout"
	"net/url"
)

type WebhookMiddleman struct {
	handler sprout.Handler
}

func NewRegistry() *WebhookMiddleman {
	return &WebhookMiddleman{}
}

func (reg *WebhookMiddleman) UID() string {
	return "framjet/go-webhook-middleman.sprout"
}

func (reg *WebhookMiddleman) LinkHandler(fh sprout.Handler) error {
	reg.handler = fh

	return nil
}

func (reg *WebhookMiddleman) UrlEncode(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	encoded := url.QueryEscape(value)
	if encoded == "" {
		return "", fmt.Errorf("unable to encode url: %s", value)
	}

	return encoded, nil
}

func (reg *WebhookMiddleman) UrlDecode(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return "", fmt.Errorf("unable to decode url: %w", err)
	}

	return decoded, nil
}

func (reg *WebhookMiddleman) UrlParse(value string) (map[string]any, error) {
	dict := map[string]any{}
	parsedURL, err := url.Parse(value)

	if err != nil {
		return dict, fmt.Errorf("unable to parse url: %w", err)
	}

	dict["scheme"] = parsedURL.Scheme
	dict["host"] = parsedURL.Host
	dict["hostname"] = parsedURL.Hostname()
	dict["path"] = parsedURL.Path
	dict["rawQuery"] = parsedURL.RawQuery
	dict["query"] = parsedURL.Query()
	dict["opaque"] = parsedURL.Opaque
	dict["fragment"] = parsedURL.Fragment
	if parsedURL.User != nil {
		dict["userinfo"] = parsedURL.User.String()
	} else {
		dict["userinfo"] = ""
	}

	return dict, nil
}

func (reg *WebhookMiddleman) RegisterFunctions(funcsMap sprout.FunctionMap) error {
	sprout.AddFunction(funcsMap, "parseUrl", reg.UrlParse)
	sprout.AddFunction(funcsMap, "urlEncode", reg.UrlEncode)
	sprout.AddFunction(funcsMap, "urlDecode", reg.UrlDecode)

	return nil
}
