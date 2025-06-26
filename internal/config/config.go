package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Destinations map[string]FlexibleDestination `yaml:"destinations"`
	Variables    map[string]string              `yaml:"variables,omitempty"`
	Routes       []Route                        `yaml:"routes"`
}

type Destination struct {
	URL    string `yaml:"url,omitempty"`
	Method string `yaml:"method,omitempty"`
	Body   string `yaml:"body,omitempty"`
}

type Route struct {
	Method       string                 `yaml:"method,omitempty"`
	Methods      []string               `yaml:"methods,omitempty"`
	Path         string                 `yaml:"path,omitempty"`
	Paths        []string               `yaml:"paths,omitempty"`
	Matchers     []Matcher              `yaml:"matchers,omitempty"`
	Destinations map[string]Destination `yaml:"destinations,omitempty"`
	Response     *RouteResponse         `yaml:"response,omitempty"`
}

type RouteResponse struct {
	Status  *RouteResponseStatus `yaml:"status,omitempty"`
	Headers *map[string]string   `yaml:"headers,omitempty"`
	Body    *string              `yaml:"body,omitempty"`
}

type RouteResponseStatus struct {
	Success *int `yaml:"success,omitempty"` // Default 200 OK
	Failure *int `yaml:"failure,omitempty"` // Default 502 Bad Gateway
}

type DestinationRef struct {
	Name    string            `yaml:",omitempty"`
	URL     string            `yaml:"url,omitempty"`
	Method  string            `yaml:"method,omitempty"`
	Body    string            `yaml:"body,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

type Matcher struct {
	Params map[string]interface{} `yaml:",inline"`
	To     FlexibleTo             `yaml:"to"`
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
