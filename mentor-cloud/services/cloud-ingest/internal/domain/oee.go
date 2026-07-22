package domain

import "math"

type OEERaw struct {
	TDisponibleTotal          float64
	TMicroparada              float64
	TParadaNoAsignada         float64
	TParadaProgramada         float64
	TParadaNoProgramada       float64
	TRefrigerio               float64
	TCapacitacionObligatoria  float64
	TMantenimientoPlanificado float64
	Conteo1                   float64
	Merma                     float64
}

type OEEResult struct {
	Disponibilidad float64
	Rendimiento    float64
	Calidad        float64
	OEE            float64
	Produccion     int
}

// CalculateOEE applies the standard OEE formula.
// velocidadUS = units/second. All time fields in seconds.
// Returns percentages clamped to [0, 100].
func CalculateOEE(raw OEERaw, velocidadUS float64) OEEResult {
	paradaObligatoria := raw.TParadaProgramada + raw.TRefrigerio +
		raw.TCapacitacionObligatoria + raw.TMantenimientoPlanificado

	tDisponible := max(raw.TDisponibleTotal-paradaObligatoria, 0)

	paradaNoObligatoria := raw.TParadaNoProgramada + raw.TParadaNoAsignada
	tOperativo := max(tDisponible-paradaNoObligatoria, 0)

	tNetoProd := max(tOperativo-raw.TMicroparada, 0)

	var tNominalProd float64
	if velocidadUS > 0 {
		tNominalProd = raw.Conteo1 / velocidadUS
	}

	tPerdidoVelocidad := max(tNetoProd-tNominalProd, 0)

	var disponibilidad, rendimiento, calidad float64

	if tDisponible > 0 {
		disponibilidad = (tOperativo / tDisponible) * 100
	}

	if tOperativo > 0 && velocidadUS > 0 {
		rendimiento = ((tOperativo - raw.TMicroparada - tPerdidoVelocidad) / tOperativo) * 100
	}

	if raw.Conteo1 > 0 {
		calidad = ((raw.Conteo1 - raw.Merma) / raw.Conteo1) * 100
	}

	disponibilidad = clamp(disponibilidad, 0, 100)
	rendimiento = clamp(rendimiento, 0, 100)
	calidad = clamp(calidad, 0, 100)

	oee := round2((disponibilidad / 100) * (rendimiento / 100) * (calidad / 100) * 100)

	return OEEResult{
		Disponibilidad: round2(disponibilidad),
		Rendimiento:    round2(rendimiento),
		Calidad:        round2(calidad),
		OEE:            oee,
		Produccion:     int(raw.Conteo1),
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
