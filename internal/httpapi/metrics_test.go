package httpapi_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestMetricsIncludeUnauthorizedRequestsWithBoundedRoute(t *testing.T) {
	a := montar(t)
	r := a.requisicao(t, http.MethodGet, "/clients/123456789", "", "")
	if r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: %d", r.StatusCode)
	}
	r = a.requisicao(t, http.MethodGet, "/metrics", "", "")
	body, _ := io.ReadAll(r.Body)
	if r.StatusCode != 200 || !strings.Contains(string(body), `route="GET /clients/{id}",status="401"} 1`) {
		t.Fatalf("metrics: %d %s", r.StatusCode, body)
	}
	if strings.Contains(string(body), "123456789") {
		t.Fatal("path parameter leaked into metric labels")
	}
}
