package domain

import "time"

type CategoryTreeNode struct {
	ID         int64               `json:"id"`
	Nombre     string              `json:"nombre"`
	Codigo     string              `json:"codigo,omitempty"`
	TipoParada string              `json:"tipo_parada,omitempty"`
	Children   []*CategoryTreeNode `json:"children,omitempty"`
}

type ProductEntry struct {
	SKU         string `json:"sku"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
}

func BuildCategoryTree(flat []CategoriaParada) []*CategoryTreeNode {
	// Clave compuesta (tipo, id) para evitar colisión de IDs entre
	// cat_programada y cat_no_programada que tienen secuencias independientes.
	type nodeKey struct {
		tipo string
		id   int64
	}

	// Deduplicar por (tipo, id)
	seen := make(map[nodeKey]struct{}, len(flat))
	deduped := make([]CategoriaParada, 0, len(flat))
	for _, c := range flat {
		k := nodeKey{c.TipoParada, c.ID}
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			deduped = append(deduped, c)
		}
	}
	flat = deduped

	nodes := make(map[nodeKey]*CategoryTreeNode, len(flat))
	for _, c := range flat {
		k := nodeKey{c.TipoParada, c.ID}
		nodes[k] = &CategoryTreeNode{
			ID:         c.ID,
			Nombre:     c.Nombre,
			Codigo:     c.Codigo,
			TipoParada: c.TipoParada,
		}
	}
	var progRoots, noprogRoots []*CategoryTreeNode
	for _, c := range flat {
		k := nodeKey{c.TipoParada, c.ID}
		node := nodes[k]
		if c.PadreID != nil {
			// El padre siempre es del mismo tipo
			parentKey := nodeKey{c.TipoParada, *c.PadreID}
			if parent, ok := nodes[parentKey]; ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		if c.TipoParada == "NO_PROGRAMADA" {
			noprogRoots = append(noprogRoots, node)
		} else {
			progRoots = append(progRoots, node)
		}
	}

	// Agrupar siempre en dos nodos raíz fijos
	var result []*CategoryTreeNode
	if len(progRoots) > 0 {
		group := &CategoryTreeNode{
			ID:         -1,
			Nombre:     "Paradas Programadas",
			TipoParada: "PROGRAMADA",
			Children:   progRoots,
		}
		propagateTipoParada(group, "PROGRAMADA")
		result = append(result, group)
	}
	if len(noprogRoots) > 0 {
		group := &CategoryTreeNode{
			ID:         -2,
			Nombre:     "Paradas No Programadas",
			TipoParada: "NO_PROGRAMADA",
			Children:   noprogRoots,
		}
		propagateTipoParada(group, "NO_PROGRAMADA")
		result = append(result, group)
	}

	// Categorías de tiempo fijas — variables OEE independientes.
	// Se muestran como entradas raíz propias (no dentro del árbol dinámico)
	// para que al asignar se establezca el stop_type correcto y alimente
	// T_REFRIGERIO / T_CAPACITACION_OBLIGATORIA / T_MANTENIMIENTO_PLANIFICADO.
	result = append(result,
		&CategoryTreeNode{ID: -10, Nombre: "Refrigerio", TipoParada: "REFRIGERIO"},
		&CategoryTreeNode{ID: -11, Nombre: "Capacitación Obligatoria", TipoParada: "CAPACITACION"},
		&CategoryTreeNode{ID: -12, Nombre: "Mantenimiento Planificado", TipoParada: "MANTENIMIENTO"},
	)
	return result
}

// propagateTipoParada rellena tipo_parada en los nodos hijos que tienen el campo vacío,
// usando el valor del ancestro más cercano que sí lo tiene.
func propagateTipoParada(node *CategoryTreeNode, inherited string) {
	if node.TipoParada == "" {
		node.TipoParada = inherited
	}
	for _, child := range node.Children {
		propagateTipoParada(child, node.TipoParada)
	}
}

type CategoriaParada struct {
	ID         int64     `json:"id"`
	Nombre     string    `json:"nombre"`
	Codigo     string    `json:"codigo"`
	PadreID    *int64    `json:"padre_id,omitempty"`
	EmpresaID  *int      `json:"empresa_id,omitempty"`
	LineaID    *int      `json:"linea_id,omitempty"`
	Orden      int       `json:"orden"`
	TipoParada string    `json:"tipo_parada,omitempty"`
	Activo     bool      `json:"activo"`
	SyncedAt   time.Time `json:"synced_at"`
}

type Producto struct {
	ID          int       `json:"id"`
	Codigo      string    `json:"codigo"`
	Nombre      string    `json:"nombre"`
	EmpresaID   *int      `json:"empresa_id,omitempty"`
	Activo      bool      `json:"activo"`
	LineaID     *int      `json:"linea_id,omitempty"`
	VelocidadUS float64   `json:"velocidad_us"`
	FactorConv  int       `json:"factor_conv"`
	SyncedAt    time.Time `json:"synced_at"`
}

type Turno struct {
	ID         int       `json:"id"`
	Nombre     string    `json:"nombre"`
	HoraInicio string    `json:"hora_inicio"`
	HoraFin    string    `json:"hora_fin"`
	PlantaID   *int      `json:"planta_id,omitempty"`
	Activo     bool      `json:"activo"`
	SyncedAt   time.Time `json:"synced_at"`
}

type Usuario struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Nombre       string    `json:"nombre"`
	Apellido     string    `json:"apellido"`
	PasswordHash string    `json:"password_hash"`
	RolID        *int      `json:"rol_id,omitempty"`
	Rol          string    `json:"rol"`
	EmpresaID    int       `json:"empresa_id"`
	Activo       bool      `json:"activo"`
	SyncedAt     time.Time `json:"synced_at"`
}

type Variable struct {
	ID            int       `json:"id"`
	Nombre        string    `json:"nombre"`
	Clave         string    `json:"clave"`
	Valor         string    `json:"valor"`
	Tipo          string    `json:"tipo"`
	DispositivoID *int      `json:"dispositivo_id,omitempty"`
	PlantaID      *int      `json:"planta_id,omitempty"`
	EmpresaID     int       `json:"empresa_id"`
	Activo        bool      `json:"activo"`
	SyncedAt      time.Time `json:"synced_at"`
}

type LineaProductoVar struct {
	ID         int       `json:"id"`
	LineaID    int       `json:"linea_id"`
	VariableID int       `json:"variable_id"`
	NombreCol  string    `json:"nombre_col"`
	Orden      int       `json:"orden"`
	SyncedAt   time.Time `json:"synced_at"`
}

type ProductoCaracteristica struct {
	ID         int       `json:"id"`
	ProductoID int       `json:"producto_id"`
	LineaID    int       `json:"linea_id"`
	VariableID int       `json:"variable_id"`
	Valor      string    `json:"valor"`
	SyncedAt   time.Time `json:"synced_at"`
}

type Planta struct {
	ID            int       `json:"id"`
	Nombre        string    `json:"nombre"`
	EmpresaID     int       `json:"empresa_id"`
	EmpresaNombre string    `json:"empresa_nombre"`
	Activo        bool      `json:"activo"`
	SyncedAt      time.Time `json:"synced_at"`
}

type Linea struct {
	ID       int       `json:"id"`
	Nombre   string    `json:"nombre"`
	PlantaID int       `json:"planta_id"`
	Tipo     string    `json:"tipo"`
	Subtipo  string    `json:"subtipo"`
	Activo   bool      `json:"activo"`
	SyncedAt time.Time `json:"synced_at"`
}

type VelocidadNominal struct {
	ID          int       `json:"id"`
	LineaID     int       `json:"linea_id"`
	ProductoID  int       `json:"producto_id"`
	VelocidadUs float64   `json:"velocidad_us"`
	FactorConv  int       `json:"factor_conv"`
	SyncedAt    time.Time `json:"synced_at"`
}

type VelocidadNominalLog struct {
	ID                  int64     `json:"id"`
	ProductoID          int       `json:"producto_id"`
	SKU                 string    `json:"sku"`
	VelocidadUSAnterior *float64  `json:"velocidad_us_anterior"`
	VelocidadUSNueva    float64   `json:"velocidad_us_nueva"`
	FactorConvAnterior  *int      `json:"factor_conv_anterior"`
	FactorConvNueva     int       `json:"factor_conv_nueva"`
	Motivo              *string   `json:"motivo"`
	Usuario             *string   `json:"usuario"`
	Origen              string    `json:"origen"`
	CambiadoEn          time.Time `json:"cambiado_en"`
}

type MotivoVelocidad struct {
	ID     int    `json:"id"`
	Texto  string `json:"texto"`
	Activo bool   `json:"activo"`
	Orden  int    `json:"orden"`
}

// ProductCharacteristicsResponse is the DTO for the product-characteristics endpoint.
type ProductCharacteristicsResponse struct {
	Columnas  []ProductColumn              `json:"columnas"`
	Productos []ProductWithCharacteristics `json:"productos"`
}

type ProductColumn struct {
	VariableID int    `json:"variable_id"`
	Nombre     string `json:"nombre"`
	Orden      int    `json:"orden"`
}

type ProductWithCharacteristics struct {
	ProductoID int            `json:"producto_id"`
	Codigo     string         `json:"codigo"`
	Nombre     string         `json:"nombre"`
	Valores    map[int]string `json:"valores"`
}
