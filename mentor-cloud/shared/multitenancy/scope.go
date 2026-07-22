package multitenancy

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Scope struct {
	PlantaID int
	LineaID  int
}

type ctxKey struct{}

func WithScope(ctx context.Context, s Scope) context.Context {
	return context.WithValue(ctx, ctxKey{}, s)
}

func ScopeFrom(ctx context.Context) (Scope, bool) {
	s, ok := ctx.Value(ctxKey{}).(Scope)
	return s, ok
}

func Table(lineaID int, name string) string {
	return fmt.Sprintf("linea_%d.%s", lineaID, name)
}

func ScopeFromRequest(r *http.Request) (Scope, bool) {
	plantaID, _ := strconv.Atoi(r.Header.Get("X-Planta-ID"))
	lineaID, _ := strconv.Atoi(r.Header.Get("X-Linea-ID"))
	if plantaID > 0 && lineaID > 0 {
		return Scope{PlantaID: plantaID, LineaID: lineaID}, true
	}
	plantaID, _ = strconv.Atoi(r.URL.Query().Get("planta_id"))
	lineaID, _ = strconv.Atoi(r.URL.Query().Get("linea_id"))
	if plantaID > 0 && lineaID > 0 {
		return Scope{PlantaID: plantaID, LineaID: lineaID}, true
	}
	return Scope{}, false
}

func Tbl(schema, legacyQualified string) string {
	if schema == "" {
		return legacyQualified
	}
	parts := strings.SplitN(legacyQualified, ".", 2)
	name := parts[len(parts)-1]
	return schema + "." + name
}
