package domain

import "time"

type LineConfig struct {
	ID            int           `json:"id"`
	DeviceID      string        `json:"device_id"`
	ConfigVersion int           `json:"config_version"`
	LineaID       int           `json:"linea_id"`
	EmpresaID     int           `json:"empresa_id"`
	ROI           ROIData       `json:"roi"`
	ROIPresencia  *ROIData      `json:"roi_presencia,omitempty"`
	Thresholds    Thresholds    `json:"thresholds"`
	FSM           FSMConfig     `json:"fsm"`
	Mode          string        `json:"mode"`
	Camera        *Camera       `json:"camera,omitempty"`
	OEE           OEEConfig     `json:"oee"`
	Cloud         CloudConfig   `json:"cloud"`
	Tablet        *TabletConfig `json:"tablet,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type ROIData struct {
	X            int `json:"x"`
	Y            int `json:"y"`
	Width        int `json:"width"`
	Height       int `json:"height"`
	BottomMargin int `json:"bottom_margin,omitempty"`
}

type Thresholds struct {
	Edge  float64 `json:"edge"`
	Color float64 `json:"color"`
	// Flow actúa como gate de fusión: si flow_signal < Flow → score total = 0
	Flow                    float64        `json:"flow"`
	DY                      float64        `json:"dy"`
	Beige                   float64        `json:"beige"`
	High                    float64        `json:"high"`
	Low                     float64        `json:"low"`
	BeigeRange              *BeigeHSVRange `json:"beige_range,omitempty"`
	PresenciaThreshold      float64        `json:"presencia_threshold,omitempty"`
	PresenciaScale          float64        `json:"presencia_scale,omitempty"`
	PresenciaHold           int            `json:"presencia_hold,omitempty"`
	PresenciaWindow         int            `json:"presencia_window,omitempty"`
	PresenciaPixelThreshold float64        `json:"presencia_pixel_threshold,omitempty"`
	PresenciaReset          bool           `json:"presencia_reset,omitempty"`
}

type BeigeHSVRange struct {
	HMin int `json:"h_min"`
	HMax int `json:"h_max"`
	SMin int `json:"s_min"`
	SMax int `json:"s_max"`
	VMin int `json:"v_min"`
}

type FSMConfig struct {
	NFrames           int     `json:"n_frames"`
	Cooldown          int     `json:"cooldown"`
	ExitFrames        int     `json:"exit_frames"`
	MaxWaitExitFrames int     `json:"max_wait_exit_frames"`
	MinRearmS         float64 `json:"min_rearm_s,omitempty"` // min seconds between two CORTEs (0=disabled)
}

type Camera struct {
	URL          string  `json:"url"`
	FPS          int     `json:"fps"`
	FrameBackend string  `json:"frame_backend,omitempty"`
	FrameSkip    int     `json:"frame_skip,omitempty"`
	SignalScale  float64 `json:"signal_scale,omitempty"`
}

type OEEConfig struct {
	LineName       string  `json:"line_name"`
	MicroStopMaxS  float64 `json:"micro_stop_max_s"`
	StopMaxS       float64 `json:"stop_max_s"`
	SnapshotInterS float64 `json:"snapshot_interval_s"`
	// VelUnit indica la unidad de entrada del operador: "us" (unidades/segundo) o "uh" (unidades/hora).
	// VelNominalUS SIEMPRE se almacena en u/s internamente — la conversión ocurre en la UI.
	VelUnit      string  `json:"vel_unit,omitempty"`
	VelNominalUS float64 `json:"vel_nominal_us,omitempty"`
}

type CloudConfig struct {
	SyncIntervalS int    `json:"sync_interval_s"`
	CloudURL      string `json:"cloud_url,omitempty"`
	CloudAPIKey   string `json:"cloud_api_key,omitempty"`
}

type TabletConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	GatewayPort int    `json:"gatewayPort"`
}

// ModeDefaultOEE devuelve los parámetros OEE canónicos para detección textil.
//
// Clasificación binaria:
//   - parada continua < micro_stop_max_s → T_MICROPARADA (descartada, no justificable)
//   - parada continua ≥ micro_stop_max_s → T_PARADA_NO_ASIGNADA (el operario justifica)
//
// stop_max_s = 86400 s (centinela: T_PARADA_MAYOR nunca se activa).
func ModeDefaultOEE(mode string) OEEConfig {
	return OEEConfig{
		MicroStopMaxS:  120,
		StopMaxS:       86400,
		SnapshotInterS: 1800,
		VelUnit:        "uh",
		VelNominalUS:   0.008333333,
	}
}

func (c *LineConfig) Validate() error {
	if c.ROI.Width <= 0 || c.ROI.Height <= 0 {
		return ErrInvalidROI
	}

	if c.Thresholds.Edge < 0 || c.Thresholds.Edge > 1 {
		return ErrInvalidThreshold
	}

	if c.Thresholds.Color < 0 || c.Thresholds.Color > 1 {
		return ErrInvalidThreshold
	}

	if c.Thresholds.Flow < 0 || c.Thresholds.Flow > 1 {
		return ErrInvalidThreshold
	}

	if c.Thresholds.Beige < 0 || c.Thresholds.Beige > 1 {
		return ErrInvalidThreshold
	}

	if c.FSM.NFrames < 1 || c.FSM.NFrames > 30 {
		return ErrInvalidFSM
	}

	if c.FSM.Cooldown < 0 || c.FSM.Cooldown > 60 {
		return ErrInvalidFSM
	}

	if c.FSM.ExitFrames < 0 || c.FSM.ExitFrames > 100 {
		return ErrInvalidFSM
	}

	if c.FSM.MaxWaitExitFrames < 0 || c.FSM.MaxWaitExitFrames > 15000 {
		return ErrInvalidFSM
	}

	if c.FSM.MinRearmS < 0 || c.FSM.MinRearmS > 300 {
		return ErrInvalidFSM
	}

	if c.Mode != "textil" {
		return ErrInvalidMode
	}

	if c.OEE.MicroStopMaxS < 0 || c.OEE.StopMaxS < 0 || c.OEE.SnapshotInterS < 0 {
		return ErrInvalidOEE
	}
	if c.OEE.MicroStopMaxS > 0 && c.OEE.StopMaxS > 0 && c.OEE.MicroStopMaxS > c.OEE.StopMaxS {
		return ErrInvalidOEE
	}

	if c.Cloud.SyncIntervalS < 0 {
		return ErrInvalidCloud
	}

	return nil
}
