package adapters

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"cloud-gateway/internal/ports"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey int

const ctxMeta ctxKey = iota

// RequestMeta is populated by auth middlewares and read by the router for audit logging.
type RequestMeta struct {
	UserID    int64
	EmpresaID int64
	DeviceID  string
}

type Middleware struct {
	jwtSecret   []byte
	globalKey   string
	internalKey string
	deviceRepo  ports.DeviceRepository
	rlEdge      *RateLimiter
	rlUser      *RateLimiter
	origins     map[string]bool
	allowAll    bool
}

func NewMiddleware(jwtSecret, globalKey, internalKey string, deviceRepo ports.DeviceRepository, origins []string) *Middleware {
	all := false
	set := make(map[string]bool, len(origins))
	for _, o := range origins {
		if o == "*" {
			all = true
		} else {
			set[o] = true
		}
	}
	return &Middleware{
		jwtSecret:   []byte(jwtSecret),
		globalKey:   globalKey,
		internalKey: internalKey,
		deviceRepo:  deviceRepo,
		rlEdge:      NewRateLimiter(20, 10),
		rlUser:      NewRateLimiter(50, 20),
		origins:     set,
		allowAll:    all,
	}
}

func (m *Middleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if m.allowAll || m.origins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers",
			"Authorization, Content-Type, X-API-Key, X-Device-ID, X-Internal-Key, X-Actor")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "43200")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) JWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") {
			jsonError(w, "token requerido", http.StatusUnauthorized)
			return
		}
		token, err := jwt.Parse(strings.TrimPrefix(raw, "Bearer "),
			func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return m.jwtSecret, nil
			})
		if err != nil || !token.Valid {
			jsonError(w, "token invalido", http.StatusUnauthorized)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			jsonError(w, "claims invalidos", http.StatusUnauthorized)
			return
		}
		userID := int64(asFloat(claims["sub"]))
		empresaID := int64(asFloat(claims["empresa_id"]))
		role, _ := claims["role"].(string)

		if !m.rlUser.Allow(strconv.FormatInt(userID, 10)) {
			jsonError(w, "rate limit excedido", http.StatusTooManyRequests)
			return
		}
		if meta, ok := r.Context().Value(ctxMeta).(*RequestMeta); ok {
			meta.UserID = userID
			meta.EmpresaID = empresaID
		}
		r.Header.Set("X-User-ID", strconv.FormatInt(userID, 10))
		r.Header.Set("X-Empresa-ID", strconv.FormatInt(empresaID, 10))
		r.Header.Set("X-Role", role)
		next.ServeHTTP(w, r)
	})
}

// JWTStream is like JWT but also accepts the token via the ?token= query param.
// This allows <img src="/api/v1/camera/stream?token=..."> to authenticate in
// browsers that cannot set Authorization headers on media elements.
func (m *Middleware) JWTStream(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") {
			if t := r.URL.Query().Get("token"); t != "" {
				raw = "Bearer " + t
			}
		}
		if !strings.HasPrefix(raw, "Bearer ") {
			jsonError(w, "token requerido", http.StatusUnauthorized)
			return
		}
		token, err := jwt.Parse(strings.TrimPrefix(raw, "Bearer "),
			func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return m.jwtSecret, nil
			})
		if err != nil || !token.Valid {
			jsonError(w, "token invalido", http.StatusUnauthorized)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			jsonError(w, "claims invalidos", http.StatusUnauthorized)
			return
		}
		userID := int64(asFloat(claims["sub"]))
		empresaID := int64(asFloat(claims["empresa_id"]))
		role, _ := claims["role"].(string)

		r.Header.Set("X-User-ID", strconv.FormatInt(userID, 10))
		r.Header.Set("X-Empresa-ID", strconv.FormatInt(empresaID, 10))
		r.Header.Set("X-Role", role)
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) APIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := extractKey(r)
		if key == "" {
			jsonError(w, "api key requerida", http.StatusUnauthorized)
			return
		}
		deviceID := r.Header.Get("X-Device-ID")

		if key == m.globalKey {
			if !m.rlEdge.Allow(deviceID) {
				jsonError(w, "rate limit excedido", http.StatusTooManyRequests)
				return
			}
			if meta, ok := r.Context().Value(ctxMeta).(*RequestMeta); ok {
				meta.DeviceID = deviceID
			}
			if deviceID != "" {
				go m.deviceRepo.UpdateLastSeen(context.Background(), deviceID)
			}
			next.ServeHTTP(w, r)
			return
		}

		device, err := m.deviceRepo.FindByAPIKey(r.Context(), key)
		if err != nil || device == nil {
			jsonError(w, "api key invalida", http.StatusForbidden)
			return
		}
		if !m.rlEdge.Allow(device.DeviceID) {
			jsonError(w, "rate limit excedido", http.StatusTooManyRequests)
			return
		}
		if meta, ok := r.Context().Value(ctxMeta).(*RequestMeta); ok {
			meta.DeviceID = device.DeviceID
			meta.EmpresaID = device.EmpresaID
		}
		go m.deviceRepo.UpdateLastSeen(context.Background(), device.DeviceID)
		r.Header.Set("X-Device-ID", device.DeviceID)
		r.Header.Set("X-Empresa-ID", strconv.FormatInt(device.EmpresaID, 10))
		r.Header.Set("X-Planta-ID", strconv.FormatInt(device.PlantaID, 10))
		r.Header.Set("X-Linea-ID", strconv.FormatInt(device.LineaID, 10))
		// Replace device-specific key with global key for downstream services
		r.Header.Set("Authorization", "Bearer "+m.globalKey)
		r.Header.Del("X-API-Key")
		next.ServeHTTP(w, r)
	})
}

func extractKey(r *http.Request) string {
	if k := r.Header.Get("X-API-Key"); k != "" {
		return k
	}
	if raw := r.Header.Get("Authorization"); strings.HasPrefix(raw, "Bearer ") {
		return strings.TrimPrefix(raw, "Bearer ")
	}
	return ""
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}

func asFloat(v any) float64 {
	f, _ := v.(float64)
	return f
}

func (m *Middleware) InternalKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Internal-Key")
		if key == "" || key != m.internalKey {
			jsonError(w, "internal key invalida", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var (
	reDate         = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?)?$`)
	reUUID         = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	reTenantHeader = []string{"X-Empresa-ID", "X-Planta-ID", "X-Linea-ID", "X-User-ID", "X-Role"}
	uuidParams     = []string{"stop_id", "run_id"}
	stringParams   = []string{"reason", "nombre", "descripcion", "responsable", "category"}
)

// SanitizeHeaders elimina headers de contexto de tenant que puedan venir del cliente.
// Debe ejecutarse antes de JWT y APIKey.
func (m *Middleware) SanitizeHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, h := range reTenantHeader {
			r.Header.Del(h)
		}
		next.ServeHTTP(w, r)
	})
}

// ScopeEnforcer verifica que empresa_id/planta_id/linea_id en query params y body
// coincidan con el scope del JWT. Debe ejecutarse despues de JWT.
func (m *Middleware) ScopeEnforcer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ADMIN y superadmin pueden acceder a recursos de cualquier empresa
		role := r.Header.Get("X-Role")
		if role == "ADMIN" || role == "superadmin" {
			next.ServeHTTP(w, r)
			return
		}

		jwtEmpresaID := r.Header.Get("X-Empresa-ID")
		if jwtEmpresaID == "" || jwtEmpresaID == "0" {
			next.ServeHTTP(w, r)
			return
		}

		q := r.URL.Query()
		if v := q.Get("empresa_id"); v != "" && v != jwtEmpresaID {
			jsonError(w, "acceso denegado", http.StatusForbidden)
			return
		}

		if v := q.Get("planta_id"); v != "" {
			if !m.plantaBelongsToEmpresa(r.Context(), v, jwtEmpresaID) {
				jsonError(w, "acceso denegado", http.StatusForbidden)
				return
			}
		}

		if v := q.Get("linea_id"); v != "" {
			if !m.lineaBelongsToEmpresa(r.Context(), v, jwtEmpresaID) {
				jsonError(w, "acceso denegado", http.StatusForbidden)
				return
			}
		}

		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			if ct := r.Header.Get("Content-Type"); strings.Contains(ct, "application/json") && r.Body != nil {
				body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
				if err != nil {
					jsonError(w, "error leyendo body", http.StatusBadRequest)
					return
				}
				r.Body = io.NopCloser(strings.NewReader(string(body)))

				var payload map[string]any
				if json.Unmarshal(body, &payload) == nil {
					if v, ok := payload["empresa_id"]; ok {
						if numStr := anyToStr(v); numStr != "" && numStr != jwtEmpresaID {
							jsonError(w, "acceso denegado", http.StatusForbidden)
							return
						}
					}
					if v, ok := payload["planta_id"]; ok {
						if numStr := anyToStr(v); numStr != "" {
							if !m.plantaBelongsToEmpresa(r.Context(), numStr, jwtEmpresaID) {
								jsonError(w, "acceso denegado", http.StatusForbidden)
								return
							}
						}
					}
					if v, ok := payload["linea_id"]; ok {
						if numStr := anyToStr(v); numStr != "" {
							if !m.lineaBelongsToEmpresa(r.Context(), numStr, jwtEmpresaID) {
								jsonError(w, "acceso denegado", http.StatusForbidden)
								return
							}
						}
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) plantaBelongsToEmpresa(ctx context.Context, plantaID, empresaID string) bool {
	pID, err := strconv.ParseInt(plantaID, 10, 64)
	if err != nil {
		return false
	}
	eID, err := strconv.ParseInt(empresaID, 10, 64)
	if err != nil {
		return false
	}
	return m.deviceRepo.PlantaBelongsToEmpresa(ctx, pID, eID)
}

func (m *Middleware) lineaBelongsToEmpresa(ctx context.Context, lineaID, empresaID string) bool {
	lID, err := strconv.ParseInt(lineaID, 10, 64)
	if err != nil {
		return false
	}
	eID, err := strconv.ParseInt(empresaID, 10, 64)
	if err != nil {
		return false
	}
	return m.deviceRepo.LineaBelongsToEmpresa(ctx, lID, eID)
}

func anyToStr(v any) string {
	switch val := v.(type) {
	case float64:
		return strconv.FormatInt(int64(val), 10)
	case string:
		return val
	default:
		return ""
	}
}

// ValidateInput valida parámetros de entrada comunes antes de hacer proxy.
func (m *Middleware) ValidateInput(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		for _, param := range []string{"desde", "hasta", "fecha", "from", "to"} {
			if v := q.Get(param); v != "" && !reDate.MatchString(v) {
				jsonError(w, "formato de fecha invalido: use YYYY-MM-DD", http.StatusBadRequest)
				return
			}
		}

		if v := q.Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > 5000 {
				jsonError(w, "limit debe ser un entero entre 1 y 5000", http.StatusBadRequest)
				return
			}
		}

		for _, param := range uuidParams {
			if v := q.Get(param); v != "" && !reUUID.MatchString(v) {
				jsonError(w, "formato UUID invalido en "+param, http.StatusBadRequest)
				return
			}
		}

		for _, param := range stringParams {
			if v := q.Get(param); len(v) > 200 {
				jsonError(w, param+" excede 200 caracteres", http.StatusBadRequest)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
