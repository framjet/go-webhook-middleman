package config

import (
	"fmt"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Destinations map[string]FlexibleDestination `yaml:"destinations" expr:"destinations"`
	Variables    map[string]string              `yaml:"variables,omitempty" expr:"variables"`
	Routes       []Route                        `yaml:"routes" expr:"routes"`
}

type Destination struct {
	URL    string `yaml:"url,omitempty" expr:"url"`
	Method string `yaml:"method,omitempty" expr:"method"`
	Body   string `yaml:"body,omitempty" expr:"body"`
}

type Route struct {
	Method       string                  `yaml:"method,omitempty" expr:"method"`
	Methods      []string                `yaml:"methods,omitempty" expr:"methods"`
	Path         string                  `yaml:"path,omitempty" expr:"path"`
	Paths        []string                `yaml:"paths,omitempty" expr:"paths"`
	Matchers     []*Matcher              `yaml:"matchers,omitempty" expr:"matchers"`
	Destinations map[string]*Destination `yaml:"destinations,omitempty" expr:"destinations"`
	Response     *RouteResponse          `yaml:"response,omitempty" expr:"response"`
}

type RouteResponse struct {
	Status  *RouteResponseStatus `yaml:"status,omitempty" expr:"status"`
	Headers *map[string]string   `yaml:"headers,omitempty" expr:"headers"`
	Body    *string              `yaml:"body,omitempty" expr:"body"`
}

type RouteResponseStatus struct {
	Success *int `yaml:"success,omitempty" expr:"success"` // Default 200 OK
	Failure *int `yaml:"failure,omitempty" expr:"failure"` // Default 502 Bad Gateway
}

type DestinationRef struct {
	Name    string            `yaml:"name,omitempty" expr:"name"`
	URL     string            `yaml:"url,omitempty" expr:"url"`
	Method  string            `yaml:"method,omitempty" expr:"method"`
	Body    string            `yaml:"body,omitempty" expr:"body"`
	Headers map[string]string `yaml:"headers,omitempty" expr:"headers"`
}

type Matcher struct {
	Expr     string        `yaml:"expr,omitempty" expr:"expr"`
	Exprs    []string      `yaml:"exprs,omitempty" expr:"exprs"`
	To       FlexibleTo    `yaml:"to" expr:"to"`
	programs []*vm.Program `yaml:"-"` // Compiled expressions
}

// FlexibleTo can handle both string and []DestinationRef
type FlexibleTo []DestinationRef

// FlexibleDestination can handle both string and Destination
type FlexibleDestination Destination

type ForwardResult struct {
	Destination string            `json:"destination"`
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers,omitempty"`
	Success     bool              `json:"success"`
	StatusCode  int               `json:"status_code,omitempty"`
	Error       string            `json:"error,omitempty"`
	Duration    int64             `json:"duration_ms"`
}

type ResolvedDestination struct {
	Name    string
	URL     string
	Method  string
	Headers map[string]string
	Body    []byte
}

type RequestUrlData struct {
	Full     string              `json:"full" expr:"full"`
	Scheme   string              `json:"scheme" expr:"scheme"`
	Host     string              `json:"host" expr:"host"`
	Path     string              `json:"path" expr:"path"`
	Query    map[string][]string `json:"query" expr:"query"`
	Opaque   string              `json:"opaque" expr:"opaque"`
	Fragment string              `json:"fragment" expr:"fragment"`
	UserInfo string              `json:"userInfo" expr:"userInfo"`
	RawQuery string              `json:"rawQuery" expr:"rawQuery"`
}

type RequestData struct {
	Method      string              `json:"method" expr:"method"`
	Url         RequestUrlData      `json:"url" expr:"url"`
	Headers     map[string][]string `json:"headers" expr:"headers"`
	Host        string              `json:"host" expr:"host"`
	Body        string              `json:"body" expr:"body"`
	ContentType string              `json:"contentType" expr:"contentType"`
	UserAgent   string              `json:"userAgent" expr:"userAgent"`
	RemoteAddr  string              `json:"remoteAddr" expr:"remoteAddr"`
}

type MatcherEnv struct {
	Params  map[string]string `json:"params" expr:"params"`
	Var     map[string]string `json:"var" expr:"var"`
	Matcher Matcher           `json:"matcher" expr:"matcher"`
	Config  Config            `json:"config" expr:"config"`
	Route   Route             `json:"route" expr:"route"`
	Request RequestData       `json:"request" expr:"request"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	err = config.Validate()
	if err != nil {
		return nil, err
	}

	err = config.CompileConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to compile config: %w", err)
	}

	return &config, nil
}

func (c *Config) Validate() error {
	if len(c.Routes) == 0 {
		return fmt.Errorf("no routes configured")
	}

	for i, route := range c.Routes {
		if len(route.Paths) == 0 && route.Path == "" {
			return fmt.Errorf("route %d has no paths configured", i)
		}

		for j, matcher := range route.Matchers {
			if len(matcher.To) == 0 {
				return fmt.Errorf("route %d matcher %d has no destinations", i, j)
			}
		}
	}

	// Validate destination URLs
	for name, dest := range c.Destinations {
		if dest.URL == "" {
			return fmt.Errorf("destination %s has empty URL", name)
		}
	}

	return nil
}

func (c *Config) CompileConfig() error {
	for routeIndex, route := range c.Routes {
		for matcherIndex, matcher := range route.Matchers {
			if len(matcher.To) == 0 {
				return fmt.Errorf("route %d matcher %d has no destinations", routeIndex, matcherIndex)
			}

			expressions := matcher.Exprs
			if matcher.Expr != "" {
				expressions = append(expressions, matcher.Expr)
			}

			matcher.Exprs = expressions

			err := matcher.CompileExpressions()
			if err != nil {
				return fmt.Errorf("failed to compile expressions for route %d matcher %d: %w", routeIndex, matcherIndex, err)
			}
		}
	}

	return nil
}

func (fd *FlexibleDestination) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		// Handle a single string case
		fd.URL = node.Value
		return nil
	case yaml.MappingNode:
		// Handle an object case
		var dest Destination
		if err := node.Decode(&dest); err != nil {
			return err
		}
		*fd = FlexibleDestination(dest)
		return nil
	default:
		return fmt.Errorf("invalid type for destination")
	}
}

// UnmarshalYAML implements custom unmarshalling for FlexibleTo
func (ft *FlexibleTo) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		// Handle a single string case
		*ft = FlexibleTo{DestinationRef{Name: node.Value}}
		return nil
	case yaml.SequenceNode:
		// Handle an array case
		var destinations []DestinationRef
		for _, item := range node.Content {
			var dest DestinationRef
			switch item.Kind {
			case yaml.ScalarNode:
				// String item in an array
				dest = DestinationRef{Name: item.Value}
			case yaml.MappingNode:
				// Object item in an array
				if err := item.Decode(&dest); err != nil {
					return err
				}
			default:
				return fmt.Errorf("invalid destination type in array")
			}
			destinations = append(destinations, dest)
		}
		*ft = FlexibleTo(destinations)
		return nil
	default:
		return fmt.Errorf("invalid type for 'to' field")
	}
}

func (m *Matcher) CompileExpressions() error {
	if len(m.Exprs) == 0 && m.Expr == "" {
		return fmt.Errorf("matcher has no expressions")
	}

	m.programs = make([]*vm.Program, 0, len(m.Exprs)+1)

	for _, expression := range m.Exprs {
		program, err := expr.Compile(expression, expr.Env(MatcherEnv{}))
		if err != nil {
			return fmt.Errorf("failed to compile expression '%s': %w", expression, err)
		}
		m.programs = append(m.programs, program)
	}

	return nil
}

func (m *Matcher) Evaluate(env *MatcherEnv) (bool, error) {
	if len(m.programs) == 0 {
		return false, fmt.Errorf("no compiled expressions available for matcher")
	}

	for _, program := range m.programs {
		result, err := expr.Run(program, env)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate expression: %w", err)
		}

		if result == true {
			return true, nil
		}
	}

	return false, nil
}
