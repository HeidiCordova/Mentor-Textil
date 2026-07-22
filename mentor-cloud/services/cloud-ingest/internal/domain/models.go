package domain

import (
	"encoding/json"
	"time"
)

type Variable struct {
	ID            int    `json:"id"`
	Nombre        string `json:"nombre"`
	Clave         string `json:"clave"`
	Valor         string `json:"valor"`
	Tipo          string `json:"tipo,omitempty"`
	DispositivoID *int   `json:"dispositivo_id,omitempty"`
	PlantaID      *int   `json:"planta_id,omitempty"`
	EmpresaID     *int   `json:"empresa_id,omitempty"`
	Activo        bool   `json:"activo"`
}

// FlexStrSlice deserializa un array JSON que puede contener números y/o strings
// a un []string uniforme. Esto permite compatibilidad cuando el enviador
// envía [1800, 0, "DASD"] (viejo) o ["1800", "0", "DASD"] (nuevo).
type FlexStrSlice []string

func (f *FlexStrSlice) UnmarshalJSON(b []byte) error {
	var raws []json.RawMessage
	if err := json.Unmarshal(b, &raws); err != nil {
		return err
	}
	*f = make(FlexStrSlice, len(raws))
	for i, raw := range raws {
		var n json.Number
		if err := json.Unmarshal(raw, &n); err == nil {
			(*f)[i] = n.String()
		} else {
			var s string
			if err := json.Unmarshal(raw, &s); err == nil {
				(*f)[i] = s
			} else {
				(*f)[i] = string(raw)
			}
		}
	}
	return nil
}

type OEERecord struct {
	Code      string       `json:"code"`
	Time      int64        `json:"time"`
	DeviceID  string       `json:"device_id"`
	IntervalS int          `json:"interval_s"`
	Turno     string       `json:"turno,omitempty"`
	V         int          `json:"v"`
	Head      []string     `json:"head"`
	Data      FlexStrSlice `json:"data"`
}

type RawEvent struct {
	ID            int64           `json:"id"`
	DeviceID      string          `json:"device_id"`
	EmpresaID     *int            `json:"empresa_id,omitempty"`
	PlantaID      *int            `json:"planta_id,omitempty"`
	LineaID       *int            `json:"linea_id,omitempty"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	TimestampEdge time.Time       `json:"timestamp_edge"`
	RecibidoEn    time.Time       `json:"recibido_en"`
	Procesado     bool            `json:"procesado"`
}

type OEESnapshot struct {
	ID             int64     `json:"id"`
	DeviceID       string    `json:"device_id"`
	LineaID        *int      `json:"linea_id,omitempty"`
	PlantaID       *int      `json:"planta_id,omitempty"`
	EmpresaID      *int      `json:"empresa_id,omitempty"`
	Turno          string    `json:"turno,omitempty"`
	Fecha          string    `json:"fecha"`
	Hora           time.Time `json:"hora"`
	Disponibilidad float64   `json:"disponibilidad"`
	Rendimiento    float64   `json:"rendimiento"`
	Calidad        float64   `json:"calidad"`
	OEE            float64   `json:"oee"`
	Produccion     int       `json:"produccion"`
	EnergiaKwh     float64   `json:"energia_kwh"`
	Code           string    `json:"code,omitempty"`
	IntervalS      int       `json:"interval_s,omitempty"`
	Head           []string  `json:"head,omitempty"`
	Data           []string  `json:"data,omitempty"`
	CreadoEn       time.Time `json:"creado_en"`
}

type DeviceSyncLog struct {
	DeviceID  string `json:"device_id"`
	BatchSize int    `json:"batch_size"`
	Status    string `json:"status"`
	Mensaje   string `json:"mensaje,omitempty"`
}

type IngestBatchRequest struct {
	Records []OEERecord `json:"records"`
}

// DeviceScope holds the org-level IDs resolved by the gateway from the device registry.
type DeviceScope struct {
	EmpresaID *int
	PlantaID  *int
	LineaID   *int
}

// ProductionRunRecord represents a production run synced from an edge device.
type ProductionRunRecord struct {
	RunID      string  `json:"run_id"`
	DeviceID   string  `json:"device_id"`
	ProductoID *int    `json:"producto_id,omitempty"`
	SKU        *string `json:"sku,omitempty"`
	Nombre     *string `json:"nombre,omitempty"`
	StartedAt  string  `json:"started_at"`
	EndedAt    *string `json:"ended_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

// StopRecord represents a stop event synced from an edge device.
type StopRecord struct {
	StopID      string  `json:"stop_id"`
	DeviceID    string  `json:"device_id"`
	StopType    string  `json:"stop_type"`
	StartedAt   string  `json:"started_at"`
	EndedAt     *string `json:"ended_at,omitempty"`
	DurationMS  *int64  `json:"duration_ms,omitempty"`
	Justified   bool    `json:"justified"`
	Reason      *string `json:"reason,omitempty"`
	Category    *string `json:"category,omitempty"`
	CategoriaID *int64  `json:"categoria_id,omitempty"`
	JustifiedBy *string `json:"justified_by,omitempty"`
	JustifiedAt *string `json:"justified_at,omitempty"`
	Source      string  `json:"source"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// PendingCommand representa un comando encolado desde cloud hacia edge
// para entrega confiable en el próximo ciclo del Enviador.
type PendingCommand struct {
	ID        int64           `json:"id"`
	DeviceID  string          `json:"device_id"`
	Command   string          `json:"command"`
	Payload   json.RawMessage `json:"payload"`
	Applied   bool            `json:"applied"`
	CreatedAt time.Time       `json:"created_at"`
}
