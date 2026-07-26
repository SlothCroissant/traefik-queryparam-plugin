package traefik_queryparam_plugin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestCreateConfig(t *testing.T) {
	t.Parallel()

	config := CreateConfig()
	if config == nil {
		t.Fatal("CreateConfig() returned nil")
	}
	if config.QueryParams == nil {
		t.Fatal("CreateConfig() returned a nil QueryParams map")
	}
	if config.RemoveQueryParams == nil {
		t.Fatal("CreateConfig() returned a nil RemoveQueryParams map")
	}
}

func TestNewValidation(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	tests := []struct {
		name   string
		next   http.Handler
		config *Config
		want   string
	}{
		{
			name:   "nil next handler",
			config: &Config{QueryParams: map[string]string{"key": "value"}},
			want:   "next handler cannot be nil",
		},
		{
			name: "nil configuration",
			next: next,
			want: "configuration cannot be nil",
		},
		{
			name:   "empty parameters",
			next:   next,
			config: CreateConfig(),
			want:   "queryParams and removeQueryParams cannot both be empty",
		},
		{
			name:   "empty parameter name",
			next:   next,
			config: &Config{QueryParams: map[string]string{"": "value"}},
			want:   "query parameter name cannot be empty",
		},
		{
			name:   "empty removal parameter name",
			next:   next,
			config: &Config{RemoveQueryParams: []QueryParamRemoval{{}}},
			want:   "query parameter name cannot be empty",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			handler, err := New(context.Background(), test.next, test.config, "test")
			if err == nil {
				t.Fatal("New() returned no error")
			}
			if handler != nil {
				t.Fatalf("New() handler = %v, want nil", handler)
			}
			if err.Error() != test.want {
				t.Fatalf("New() error = %q, want %q", err, test.want)
			}
		})
	}
}

func TestServeHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		method            string
		target            string
		queryParams       map[string]string
		removeQueryParams []QueryParamRemoval
		wantPath          string
		wantQuery         url.Values
	}{
		{
			name:        "adds parameter",
			method:      http.MethodGet,
			target:      "http://example.com/resource",
			queryParams: map[string]string{"added": "value"},
			wantPath:    "/resource",
			wantQuery:   url.Values{"added": {"value"}},
		},
		{
			name:        "preserves unrelated parameter",
			method:      http.MethodGet,
			target:      "http://example.com/resource?existing=original",
			queryParams: map[string]string{"added": "value"},
			wantPath:    "/resource",
			wantQuery: url.Values{
				"added":    {"value"},
				"existing": {"original"},
			},
		},
		{
			name:        "replaces all values with configured value",
			method:      http.MethodGet,
			target:      "http://example.com/resource?replace=first&replace=second",
			queryParams: map[string]string{"replace": "configured"},
			wantPath:    "/resource",
			wantQuery:   url.Values{"replace": {"configured"}},
		},
		{
			name:              "removes all values for a parameter",
			method:            http.MethodGet,
			target:            "http://example.com/resource?keep=original&remove=first&remove=second",
			removeQueryParams: []QueryParamRemoval{{Key: "remove"}},
			wantPath:          "/resource",
			wantQuery:         url.Values{"keep": {"original"}},
		},
		{
			name:              "removes matching values for a parameter",
			method:            http.MethodGet,
			target:            "http://example.com/resource?keep=original&remove=first&remove=second&remove=first",
			removeQueryParams: []QueryParamRemoval{{Key: "remove", Value: stringPtr("first")}},
			wantPath:          "/resource",
			wantQuery: url.Values{
				"keep":   {"original"},
				"remove": {"second"},
			},
		},
		{
			name:        "applies removals before additions",
			method:      http.MethodGet,
			target:      "http://example.com/resource?replace=client&remove=first&remove=second",
			queryParams: map[string]string{"replace": "configured"},
			removeQueryParams: []QueryParamRemoval{
				{Key: "remove"},
				{Key: "replace", Value: stringPtr("client")},
			},
			wantPath:  "/resource",
			wantQuery: url.Values{"replace": {"configured"}},
		},
		{
			name:   "encodes names and values",
			method: http.MethodPost,
			target: "http://example.com/resource",
			queryParams: map[string]string{
				"special name": "space & symbols",
				"empty":        "",
			},
			wantPath: "/resource",
			wantQuery: url.Values{
				"special name": {"space & symbols"},
				"empty":        {""},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var received *http.Request
			next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				received = req
				rw.WriteHeader(http.StatusNoContent)
			})

			handler, err := New(context.Background(), next, &Config{
				QueryParams:       test.queryParams,
				RemoveQueryParams: test.removeQueryParams,
			}, "test")
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			request := httptest.NewRequest(test.method, test.target, nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
			}
			if received == nil {
				t.Fatal("next handler was not called")
			}
			if received.URL.Path != test.wantPath {
				t.Errorf("path = %q, want %q", received.URL.Path, test.wantPath)
			}
			if got := received.URL.Query(); !reflect.DeepEqual(got, test.wantQuery) {
				t.Errorf("query = %#v, want %#v", got, test.wantQuery)
			}
			if got, want := received.RequestURI, received.URL.RequestURI(); got != want {
				t.Errorf("RequestURI = %q, want %q", got, want)
			}
		})
	}
}

func TestNewCopiesConfiguration(t *testing.T) {
	t.Parallel()

	config := &Config{
		QueryParams:       map[string]string{"stable": "original"},
		RemoveQueryParams: []QueryParamRemoval{{Key: "remove", Value: stringPtr("original")}},
	}
	var received url.Values
	next := http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		received = req.URL.Query()
	})

	handler, err := New(context.Background(), next, config, "test")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	config.QueryParams["stable"] = "mutated"
	*config.RemoveQueryParams[0].Value = "mutated"

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://example.com/?remove=original&remove=mutated", nil))

	want := url.Values{"stable": {"original"}, "remove": {"mutated"}}
	if !reflect.DeepEqual(received, want) {
		t.Fatalf("query = %#v, want %#v", received, want)
	}
}

func stringPtr(value string) *string {
	return &value
}
