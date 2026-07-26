# Traefik Query Parameter Plugin

A focused Traefik HTTP middleware that adds and removes configured query
parameters on every request handled by a router.

The middleware:

- adds configured parameters when they are absent;
- replaces existing values for configured parameter names;
- removes values for configured parameter names;
- preserves every unrelated query parameter;
- relies on Go's standard URL encoding for names and values; and
- only affects routers to which the middleware is attached.

## Kubernetes Installation

This plugin is intended for Traefik installed with the Helm chart and the
Kubernetes CRD provider. Use `localPlugins` to avoid the Traefik Plugin Catalog.
The release workflow publishes an init-container image with the same tag as the
plugin release. It copies the embedded plugin source into a shared `emptyDir`
before Traefik starts:

```yaml
providers:
  kubernetesCRD:
    enabled: true

experimental:
  localPlugins:
    QueryParamsPlugin:
      moduleName: github.com/SlothCroissant/traefik-queryparam-plugin

deployment:
  additionalVolumes:
    - name: plugins-local
      emptyDir: {}
  initContainers:
    - name: fetch-queryparams-plugin
      image: ghcr.io/slothcroissant/traefik-queryparam-plugin:v1.1.0
      volumeMounts:
        - name: plugins-local
          mountPath: /plugins-local

additionalVolumeMounts:
  - name: plugins-local
    mountPath: /plugins-local/src/github.com/SlothCroissant/traefik-queryparam-plugin
```

Set the image tag to the plugin release tag you want to run. The init container
copies the plugin files directly into `/plugins-local`; Traefik mounts that same
volume at the module path it requires. This image contains no Traefik runtime; it
only stages the plugin source for the Traefik container.

Create a `Middleware` resource in the same namespace as the route that uses
it:

```yaml
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: app-query-parameters
  namespace: default
spec:
  plugin:
    QueryParamsPlugin:
      addQueryParams:
        - key: source
          value: traefik
        - key: environment
          value: production
      removeQueryParams:
        - key: temporary
        - key: source
          value: client
```

Reference the middleware from an `IngressRoute`:

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: app
  namespace: default
spec:
  entryPoints:
    - web
  routes:
    - match: Host(`app.example.com`)
      kind: Rule
      middlewares:
        - name: app-query-parameters
      services:
        - name: app
          port: 80
```

A request for `/items?page=2&source=client` is forwarded as:

```text
/items?environment=production&page=2&source=traefik
```

Each `addQueryParams` item adds or replaces a parameter with its `key` and
`value`. Each `removeQueryParams` item removes every value for its `key` when
`value` is omitted, or only matching values when `value` is set. Removals are
applied before additions. Query strings are encoded in deterministic key order.
Traefik rejects a middleware with both `addQueryParams` and
`removeQueryParams` empty.

## Local Development

For local development, register the repository as a local plugin:

```yaml
experimental:
  localPlugins:
    QueryParamsPlugin:
      moduleName: github.com/SlothCroissant/traefik-queryparam-plugin
```

Traefik expects the repository at:

```text
/plugins-local/src/github.com/SlothCroissant/traefik-queryparam-plugin
```

The included Compose integration test mounts the repository at that location.

## Validation

Run the unit tests with a local Go installation:

```sh
make test
```

Run the end-to-end test against the pinned Traefik release and a real upstream
service:

```sh
make integration-test
```

The integration test uses port `18080` by default. Override it when needed:

```sh
TRAEFIK_TEST_PORT=28080 make integration-test
```

Pull requests run formatting, `go vet`, race-enabled unit tests, and the Docker
integration test through GitHub Actions.
