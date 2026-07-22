package domain

import (
	"encoding/json"
	"time"
)

type Planta struct {
	ID            int        `json:"id"`
	Nombre        string     `json:"nombre"`
	EmpresaID     int        `json:"empresa_id,omitempty"`
	Compania      string     `json:"compania,omitempty"`
	Lineas        int        `json:"lineas"`
	Activo        bool       `json:"activo"`
	Tipo          string     `json:"tipo"`
	DBName        *string    `json:"db_name,omitempty"`
	InstanceType  *string    `json:"instance_type,omitempty"`
	DBHost        *string    `json:"db_host,omitempty"`
	DBSchemas     []int      `json:"db_schemas,omitempty"`
	ProvisionedAt *time.Time `json:"provisioned_at,omitempty"`
	RDSClass      *string    `json:"rds_class,omitempty"`
	RDSRegion     *string    `json:"rds_region,omitempty"`
	CreadoEn      time.Time  `json:"-"`
	ActualizadoEn time.Time  `json:"-"`
}

type Linea struct {
	ID       int       `json:"id"`
	Nombre   string    `json:"nombre"`
	PlantaID int       `json:"planta_id"`
	Tipo     string    `json:"tipo"`
	Subtipo  string    `json:"subtipo"`
	Mode     string    `json:"mode"`
	Activo   bool      `json:"activo"`
	CreadoEn time.Time `json:"-"`
}

type Dispositivo struct {
	ID            int        `json:"id"`
	DeviceID      string     `json:"device_id"`
	Nombre        string     `json:"nombre"`
	LineaID       *int       `json:"linea_id,omitempty"`
	LocacionID    *int       `json:"locacion_id,omitempty"`
	PlantaID      *int       `json:"planta_id,omitempty"`
	EmpresaID     *int       `json:"empresa_id,omitempty"`
	Estado        string     `json:"estado"`
	UltimoPing    *time.Time `json:"ultimo_ping,omitempty"`
	Activo        bool       `json:"activo"`
	CreadoEn      time.Time  `json:"-"`
	ActualizadoEn time.Time  `json:"-"`
}

type DispositivoConEstado struct {
	ID            int        `json:"id"`
	DeviceID      string     `json:"device_id"`
	Nombre        string     `json:"nombre"`
	LineaID       *int       `json:"linea_id,omitempty"`
	LocacionID    *int       `json:"locacion_id,omitempty"`
	PlantaID      *int       `json:"planta_id,omitempty"`
	EmpresaID     *int       `json:"empresa_id,omitempty"`
	Estado        string     `json:"estado"`
	Activo        bool       `json:"activo"`
	LastSeenAt    *time.Time `json:"last_seen_at,omitempty"`
	GatewayActive bool       `json:"gateway_active"`
}

type Variable struct {
	ID            int       `json:"id"`
	Nombre        string    `json:"nombre"`
	Clave         string    `json:"clave"`
	Valor         string    `json:"valor"`
	Tipo          string    `json:"tipo"`
	DispositivoID *int      `json:"dispositivo_id,omitempty"`
	PlantaID      *int      `json:"planta_id,omitempty"`
	EmpresaID     *int      `json:"empresa_id,omitempty"`
	Activo        bool      `json:"activo"`
	CreadoEn      time.Time `json:"-"`
	ActualizadoEn time.Time `json:"-"`
}

type DeviceConfig struct {
	ID            int       `json:"id"`
	DispositivoID int       `json:"dispositivo_id"`
	Version       int       `json:"version"`
	Payload       []byte    `json:"payload"`
	Aplicado      bool      `json:"aplicado"`
	CreadoEn      time.Time `json:"creado_en"`
}

type Turno struct {
	ID         int    `json:"id"`
	Nombre     string `json:"nombre"`
	HoraInicio string `json:"hora_inicio"`
	HoraFin    string `json:"hora_fin"`
	PlantaID   *int   `json:"planta_id,omitempty"`
	Activo     bool   `json:"activo"`
}

type Producto struct {
	ID        int    `json:"id"`
	Codigo    string `json:"codigo"`
	Nombre    string `json:"nombre"`
	EmpresaID *int   `json:"empresa_id,omitempty"`
	Activo    bool   `json:"activo"`
}
type Locacion struct {
	ID          int    `json:"id"`
	Nombre      string `json:"nombre"`
	Descripcion string `json:"descripcion,omitempty"`
	LineaID     *int   `json:"linea_id,omitempty"`
	LineaNombre string `json:"linea_nombre,omitempty"`
	EmpresaID   *int   `json:"empresa_id,omitempty"`
	Activo      bool   `json:"activo"`
}

type PendingConfig struct {
	Version int             `json:"version"`
	Payload json.RawMessage `json:"payload"`
}

type ConfigAck struct {
	DeviceID string `json:"device_id"`
	Version  int    `json:"version"`
}

type ConfigSync struct {
	DeviceID string          `json:"device_id"`
	Payload  json.RawMessage `json:"payload"`
}
