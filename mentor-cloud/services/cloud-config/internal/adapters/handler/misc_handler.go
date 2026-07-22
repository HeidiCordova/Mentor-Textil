package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func empresaIDFromContext(c *gin.Context) int64 {
	if eid, exists := c.Get("empresa_id"); exists {
		if id, ok := eid.(int64); ok {
			return id
		}
	}
	id, _ := strconv.ParseInt(c.GetHeader("X-Empresa-ID"), 10, 64)
	return id
}

// empresaIDForLinea deriva el empresa_id dado un linea_id consultando la BD.
// Útil para admin sin empresa en JWT que opera sobre líneas de otras empresas.
func empresaIDForLinea(ctx context.Context, db *pgxpool.Pool, lineaID int64) int64 {
	var eid int64
	_ = db.QueryRow(ctx,
		`SELECT p.empresa_id FROM config.lineas l
		 JOIN config.plantas p ON p.id = l.planta_id
		 WHERE l.id = $1`, lineaID,
	).Scan(&eid)
	return eid
}

// resolveEmpresaID deriva siempre el empresa_id desde el linea_id en BD cuando está disponible.
// Esto garantiza broadcasts correctos también para admin cuyo JWT tiene empresa_id=1 (superadmin).
func resolveEmpresaIDWithFallback(ctx context.Context, c *gin.Context, db *pgxpool.Pool, lineaID int64) int64 {
	if lineaID > 0 {
		if eid := empresaIDForLinea(ctx, db, lineaID); eid > 0 {
			return eid
		}
	}
	return empresaIDFromContext(c)
}
