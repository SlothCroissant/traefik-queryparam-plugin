// Package traefik_queryparam_plugin provides a Traefik middleware that adds
// and removes configured query parameters from requests.
package traefik_queryparam_plugin

import (
	"context"
	"fmt"
	"net/http"
)

// Config defines the query parameters added and removed by the middleware.
type Config struct {
	AddQueryParams    map[string]string   `json:"addQueryParams,omitempty"`
	RemoveQueryParams []QueryParamRemoval `json:"removeQueryParams,omitempty"`
}

// QueryParamRemoval defines a query parameter removal rule. Omit Value to
// remove all values for Key; set Value to remove only matching values.
type QueryParamRemoval struct {
	Key   string  `json:"key"`
	Value *string `json:"value,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		AddQueryParams:    make(map[string]string),
		RemoveQueryParams: make([]QueryParamRemoval, 0),
	}
}

// QueryParam applies configured query parameter changes before forwarding a request.
type QueryParam struct {
	next              http.Handler
	addQueryParams    map[string]string
	removeQueryParams []QueryParamRemoval
}

// New creates a query parameter middleware.
func New(_ context.Context, next http.Handler, config *Config, _ string) (http.Handler, error) {
	if next == nil {
		return nil, fmt.Errorf("next handler cannot be nil")
	}
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	if len(config.AddQueryParams) == 0 && len(config.RemoveQueryParams) == 0 {
		return nil, fmt.Errorf("addQueryParams and removeQueryParams cannot both be empty")
	}

	addQueryParams := make(map[string]string, len(config.AddQueryParams))
	for key, value := range config.AddQueryParams {
		if key == "" {
			return nil, fmt.Errorf("query parameter name cannot be empty")
		}
		addQueryParams[key] = value
	}

	removeQueryParams := make([]QueryParamRemoval, len(config.RemoveQueryParams))
	for i, removal := range config.RemoveQueryParams {
		if removal.Key == "" {
			return nil, fmt.Errorf("query parameter name cannot be empty")
		}
		removeQueryParams[i].Key = removal.Key
		if removal.Value != nil {
			valueCopy := *removal.Value
			removeQueryParams[i].Value = &valueCopy
		}
	}

	return &QueryParam{
		next:              next,
		addQueryParams:    addQueryParams,
		removeQueryParams: removeQueryParams,
	}, nil
}

// ServeHTTP removes configured values, adds configured values, and calls the next handler.
func (q *QueryParam) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	for _, removal := range q.removeQueryParams {
		if removal.Value == nil {
			query.Del(removal.Key)
			continue
		}

		values := query[removal.Key]
		filtered := values[:0]
		for _, candidate := range values {
			if candidate != *removal.Value {
				filtered = append(filtered, candidate)
			}
		}
		if len(filtered) == 0 {
			delete(query, removal.Key)
			continue
		}
		query[removal.Key] = filtered
	}
	for key, value := range q.addQueryParams {
		query.Set(key, value)
	}
	req.URL.RawQuery = query.Encode()
	req.RequestURI = req.URL.RequestURI()

	q.next.ServeHTTP(rw, req)
}
