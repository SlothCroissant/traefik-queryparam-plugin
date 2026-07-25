# Traefik Query Parameter Plugin

A focused Traefik HTTP middleware that guarantees configured query parameters
are present on every request handled by a router.

The middleware:

- adds configured parameters when they are absent;
- replaces existing values for configured parameter names;
- preserves every unrelated query parameter;
- relies on Go's standard URL encoding for names and values; and
- only affects routers to which the middleware is attached.

## Configuration

Register the plugin in Traefik's static configuration:

```yaml
experimental:
  plugins:
    queryparam:
      moduleName: github.com/SlothCroissant/traefik-queryparam-plugin
      version: v0.1.0
```

Then define and attach the middleware in dynamic configuration:

```yaml
http:
  routers:
    app:
      rule: Host(`app.example.com`)
      middlewares:
        - app-query-parameters
      service: app

  middlewares:
    app-query-parameters:
      plugin:
        queryparam:
          queryParams:
            source: traefik
            environment: production
```

A request for `/items?page=2&source=client` is forwarded as:

```text
/items?environment=production&page=2&source=traefik
```

Query strings are encoded in deterministic key order. An empty
`queryParams` map is rejected when Traefik creates the middleware.

## Local plugin development

For local development, register the repository as a local plugin:

```yaml
experimental:
  localPlugins:
    queryparam:
      moduleName: github.com/SlothCroissant/traefik-queryparam-plugin
```

Traefik expects the repository at:

```text
/plugins-local/src/github.com/SlothCroissant/traefik-queryparam-plugin
```

The included Compose configuration mounts the repository at that location.

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
