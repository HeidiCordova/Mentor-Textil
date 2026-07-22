package domain

import "strconv"

var headFieldMapping = map[string]string{
	// Aliases estándar (mayúsculas)
	"IA":                "corriente_a",
	"IB":                "corriente_b",
	"IC":                "corriente_c",
	"I_AVG":             "corriente_avg",
	"UA":                "voltaje_a",
	"UB":                "voltaje_b",
	"UC":                "voltaje_c",
	"U_AVG":             "voltaje_avg",
	"UAB":               "voltaje_ab",
	"UBC":               "voltaje_bc",
	"UAC":               "voltaje_ac",
	"POTENCIA_ACTIVA":   "potencia_activa",
	"POTENCIA_REACTIVA": "potencia_reactiva",
	"POTENCIA_APARENTE": "potencia_aparente",
	"FACTOR_POTENCIA":   "factor_potencia",
	"FRECUENCIA":        "frecuencia_hz",
	"ENERGIA_ACTIVA":    "energia_activa",
	"ENERGIA_REACTIVA":  "energia_reactiva",
	"ENERGIA_APARENTE":  "energia_aparente",
	"THD_IA":            "thd_ia",
	"THD_IB":            "thd_ib",
	"THD_IC":            "thd_ic",
	"THD_UA":            "thd_ua",
	"THD_UB":            "thd_ub",
	"THD_UC":            "thd_uc",
	// Aliases MC60 (mixed case tal como llegan del medidor)
	"Ia":   "corriente_a",
	"Ib":   "corriente_b",
	"Ic":   "corriente_c",
	"Iavg": "corriente_avg",
	"In":   "corriente_neutro",
	"Va":   "voltaje_a",
	"Vb":   "voltaje_b",
	"Vc":   "voltaje_c",
	"Vavg": "voltaje_avg",
	"Vab":  "voltaje_ab",
	"Vbc":  "voltaje_bc",
	// Nota: el REGISTER_MAP usa "Vca" (no "Vac")
	"Vca":  "voltaje_ac",
	"Vac":  "voltaje_ac", // alias legacy por si acaso
	"P":    "potencia_activa",
	"Q":    "potencia_reactiva",
	"S":    "potencia_aparente",
	"PF":   "factor_potencia",
	"Freq": "frecuencia_hz",
	// Energia total positiva (import) — nombres actuales del REGISTER_MAP
	"EPImp": "energia_activa",
	"EQImp": "energia_reactiva",
	"ES":    "energia_aparente",
	// Aliases legacy (versiones anteriores del firmware/reader)
	"Ea":    "energia_activa",
	"Er":    "energia_reactiva",
	"Es":    "energia_aparente",
	"THDia": "thd_ia",
	"THDib": "thd_ib",
	"THDic": "thd_ic",
	"THDua": "thd_ua",
	"THDub": "thd_ub",
	"THDuc": "thd_uc",
}

func ExtractFields(head []string, data []string) map[string]float64 {
	idx := make(map[string]int, len(head))
	for i, h := range head {
		idx[h] = i
	}

	result := make(map[string]float64, len(headFieldMapping))
	for headKey, fieldName := range headFieldMapping {
		i, ok := idx[headKey]
		if !ok || i >= len(data) {
			continue
		}
		v, err := strconv.ParseFloat(data[i], 64)
		if err != nil {
			continue
		}
		result[fieldName] = v
	}
	return result
}
