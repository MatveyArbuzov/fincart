package httpserver

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const (
	openAPIFile = "../../docs/openapi.yaml"
	apiPrefix   = "/api/v1"
)

type openAPIDocument struct {
	Paths map[string]map[string]yaml.Node `yaml:"paths"`
}

func TestOpenAPIContract(t *testing.T) {
	document := loadOpenAPIDocument(t)

	actualRoutes := make(map[string]struct{})

	for _, route := range Routes() {
		path := normalizePath(route.Path)

		key := routeKey(route.Method, path)

		actualRoutes[key] = struct{}{}
	}

	documentedRoutes := make(map[string]struct{})

	for path, methods := range document.Paths {
		for method := range methods {
			if !isHTTPMethod(method) {
				continue
			}

			key := routeKey(
				method,
				normalizePath(path),
			)

			documentedRoutes[key] = struct{}{}
		}
	}

	t.Run("HTTP routes are documented", func(t *testing.T) {
		for _, route := range Routes() {
			path := normalizePath(route.Path)

			key := routeKey(
				route.Method,
				path,
			)

			if _, ok := documentedRoutes[key]; !ok {
				t.Errorf(
					"HTTP route is missing in OpenAPI: %s %s",
					route.Method,
					route.Path,
				)
			}
		}
	})

	t.Run("OpenAPI routes exist in HTTP router", func(t *testing.T) {
		for path, methods := range document.Paths {
			for method := range methods {
				if !isHTTPMethod(method) {
					continue
				}

				normalizedPath := normalizePath(path)

				key := routeKey(
					method,
					normalizedPath,
				)

				if _, ok := actualRoutes[key]; !ok {
					t.Errorf(
						"OpenAPI route is missing in HTTP router: %s %s",
						strings.ToUpper(method),
						path,
					)
				}
			}
		}
	})
}

func loadOpenAPIDocument(t *testing.T) openAPIDocument {
	t.Helper()

	data, err := os.ReadFile(openAPIFile)
	if err != nil {
		t.Fatalf(
			"read OpenAPI document %q: %v",
			openAPIFile,
			err,
		)
	}

	var document openAPIDocument

	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatalf(
			"parse OpenAPI document %q: %v",
			openAPIFile,
			err,
		)
	}

	if document.Paths == nil {
		t.Fatalf(
			"OpenAPI document %q does not contain paths",
			openAPIFile,
		)
	}

	return document
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}

	if strings.HasPrefix(path, apiPrefix) {
		path = strings.TrimPrefix(
			path,
			apiPrefix,
		)
	}

	if path == "" {
		return "/"
	}

	return path
}

func routeKey(method, path string) string {
	return fmt.Sprintf(
		"%s %s",
		strings.ToUpper(method),
		path,
	)
}

func isHTTPMethod(method string) bool {
	switch strings.ToLower(method) {
	case
		"get",
		"put",
		"post",
		"delete",
		"options",
		"head",
		"patch",
		"trace":
		return true

	default:
		return false
	}
}
