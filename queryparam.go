// Package traefik_queryparam_plugin provides a Traefik middleware that adds
// configured query parameters to requests.
package traefik_queryparam_plugin

import (
	"context"
	"fmt"
	"net/http"
)

// Config defines the query parameters added by the middleware.
type Config struct {
	QueryParams map[string]string `json:"queryParams,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		QueryParams: make(map[string]string),
	}
}

// QueryParam adds configured query parameters before forwarding a request.
type QueryParam struct {
	next        http.Handler
	queryParams map[string]string
}

// New creates a query parameter middleware.
func New(_ context.Context, next http.Handler, config *Config, _ string) (http.Handler, error) {
	if next == nil {
		return nil, fmt.Errorf("next handler cannot be nil")
	}
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	if len(config.QueryParams) == 0 {
		return nil, fmt.Errorf("queryParams cannot be empty")
	}

	queryParams := make(map[string]string, len(config.QueryParams))
	for key, value := range config.QueryParams {
		if key == "" {
			return nil, fmt.Errorf("query parameter name cannot be empty")
		}
		queryParams[key] = value
	}

	return &QueryParam{
		next:        next,
		queryParams: queryParams,
	}, nil
}

// ServeHTTP adds the configured values and calls the next handler.
func (q *QueryParam) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	for key, value := range q.queryParams {
		query.Set(key, value)
	}
	req.URL.RawQuery = query.Encode()
	req.RequestURI = req.URL.RequestURI()

	q.next.ServeHTTP(rw, req)
}
