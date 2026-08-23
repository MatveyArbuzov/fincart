package httpserver

import (
	"os"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPIContainsAllHTTPRoutes(t *testing.T) {
	const openAPIPath = "../../docs/openapi.yaml"

	data, err := os.ReadFile(openAPIPath)
	if err != nil {
		t.Fatalf("read OpenAPI document: %v", err)
	}

	loader := openapi3.NewLoader()

	doc, err := loader.LoadFromData(data)
	if err != nil {
		t.Fatalf("load OpenAPI document: %v", err)
	}

	if err := doc.Validate(t.Context()); err != nil {
		t.Fatalf("validate OpenAPI document: %v", err)
	}

	for _, route := range Routes() {
		path := normalizeOpenAPIPath(route.Path)

		item := doc.Paths.Find(path)
		if item == nil {
			t.Errorf(
				"HTTP route %s %s is missing in OpenAPI",
				route.Method,
				route.Path,
			)
			continue
		}

		if !operationExists(item, route.Method) {
			t.Errorf(
				"HTTP route %s %s is missing in OpenAPI",
				route.Method,
				route.Path,
			)
		}
	}
}

func operationExists(
	item *openapi3.PathItem,
	method string,
) bool {
	switch strings.ToUpper(method) {
	case "GET":
		return item.Get != nil
	case "POST":
		return item.Post != nil
	case "PUT":
		return item.Put != nil
	case "PATCH":
		return item.Patch != nil
	case "DELETE":
		return item.Delete != nil
	case "HEAD":
		return item.Head != nil
	case "OPTIONS":
		return item.Options != nil
	case "TRACE":
		return item.Trace != nil
	default:
		return false
	}
}

func normalizeOpenAPIPath(path string) string {
	const prefix = "/api/v1"

	if strings.HasPrefix(path, prefix) {
		return strings.TrimPrefix(path, prefix)
	}

	return path
}
