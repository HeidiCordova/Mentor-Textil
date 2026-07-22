package adapters

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// SystemMetrics representa el estado de recursos del Jetson.
type SystemMetrics struct {
	CPUPercent    float64       `json:"cpu_percent"`
	RAMUsedMB     uint64        `json:"ram_used_mb"`
	RAMTotalMB    uint64        `json:"ram_total_mb"`
	RAMPercent    float64       `json:"ram_percent"`
	DiskUsedGB    float64       `json:"disk_used_gb"`
	DiskTotalGB   float64       `json:"disk_total_gb"`
	DiskPercent   float64       `json:"disk_percent"`
	GPUPercent    float64       `json:"gpu_percent"`
	GPUFreqMHz    int           `json:"gpu_freq_mhz"`
	GPUMaxFreqMHz int           `json:"gpu_max_freq_mhz"`
	Temps         []TempReading `json:"temps"`
}

// TempReading representa la temperatura de una zona térmica.
type TempReading struct {
	Name    string  `json:"name"`
	Celsius float64 `json:"celsius"`
}

// ── Poller de fondo ────────────────────────────────────────────────────────────

// startMetricsPoller lanza una goroutine de fondo que actualiza el caché de
// métricas cada 5 segundos. Cada ciclo tiene un timeout de 3 s para evitar
// que lecturas de /proc o /sys bloqueen indefinidamente dentro del container.
func (s *HTTPServer) startMetricsPoller() {
	go func() {
		s.updateMetricsCache() // primera lectura inmediata
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.updateMetricsCache()
		}
	}()
}

// updateMetricsCache recolecta métricas con timeout y actualiza el caché.
func (s *HTTPServer) updateMetricsCache() {
	type result struct {
		m   *SystemMetrics
		err error
	}
	ch := make(chan result, 1)
	go func() {
		m, err := collectSystemMetrics()
		ch <- result{m, err}
	}()

	select {
	case res := <-ch:
		if res.err == nil && res.m != nil {
			s.sysMetricsMu.Lock()
			s.sysMetrics = res.m
			s.sysMetricsMu.Unlock()
		}
	case <-time.After(3 * time.Second):
		log.Println("[metrics] timeout al recolectar métricas de hardware (>3s)")
	}
}

// handleSystemMetrics devuelve el caché de métricas de hardware al instante.
func (s *HTTPServer) handleSystemMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	s.sysMetricsMu.RLock()
	m := s.sysMetrics
	s.sysMetricsMu.RUnlock()

	if m == nil {
		m = &SystemMetrics{}
	}
	s.jsonResponse(w, http.StatusOK, m)
}

// ── Recolección de métricas ────────────────────────────────────────────────────

// collectSystemMetrics recolecta todas las métricas del sistema.
func collectSystemMetrics() (*SystemMetrics, error) {
	m := &SystemMetrics{}

	// ── CPU (2 muestras con 250 ms de pausa para calcular delta)
	m.CPUPercent = readCPUPercent()

	// ── RAM
	ramTotal, ramAvailable := readMemInfo()
	const toMB = 1024
	m.RAMTotalMB = ramTotal / toMB
	ramUsed := ramTotal - ramAvailable
	m.RAMUsedMB = ramUsed / toMB
	if ramTotal > 0 {
		m.RAMPercent = float64(ramUsed) / float64(ramTotal) * 100
	}

	// ── Disco (partición raíz)
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err == nil {
		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bfree * uint64(stat.Bsize)
		used := total - free
		const toGB = 1024 * 1024 * 1024
		m.DiskTotalGB = float64(total) / float64(toGB)
		m.DiskUsedGB = float64(used) / float64(toGB)
		if total > 0 {
			m.DiskPercent = float64(used) / float64(total) * 100
		}
	}

	// ── GPU carga con timeout
	m.GPUPercent = readWithTimeout(readJetsonGPU, 500*time.Millisecond)

	// ── GPU frecuencia con timeout
	m.GPUFreqMHz, m.GPUMaxFreqMHz = readGPUFreqsWithTimeout(500 * time.Millisecond)

	// ── Temperaturas con timeout
	m.Temps = readTempsWithTimeout(500 * time.Millisecond)

	return m, nil
}

// ── Helpers de timeout ─────────────────────────────────────────────────────────

func readWithTimeout(fn func() float64, timeout time.Duration) float64 {
	ch := make(chan float64, 1)
	go func() { ch <- fn() }()
	select {
	case v := <-ch:
		return v
	case <-time.After(timeout):
		return 0
	}
}

func readTempsWithTimeout(timeout time.Duration) []TempReading {
	ch := make(chan []TempReading, 1)
	go func() { ch <- readThermalZones() }()
	select {
	case v := <-ch:
		return v
	case <-time.After(timeout):
		return nil
	}
}

func readGPUFreqsWithTimeout(timeout time.Duration) (curMHz, maxMHz int) {
	type res struct{ cur, max int }
	ch := make(chan res, 1)
	go func() {
		c, mx := readJetsonGPUFreqs()
		ch <- res{c, mx}
	}()
	select {
	case v := <-ch:
		return v.cur, v.max
	case <-time.After(timeout):
		return 0, 0
	}
}

// ── Lecturas de hardware ───────────────────────────────────────────────────────

// readCPUPercent calcula el uso de CPU midiendo dos snapshots de /proc/stat
// separados 250 ms. Retorna 0 si el archivo no está disponible.
func readCPUPercent() float64 {
	snap1 := readCPUStat()
	time.Sleep(250 * time.Millisecond)
	snap2 := readCPUStat()

	if snap1 == nil || snap2 == nil {
		return 0
	}

	total1 := snap1[0] + snap1[1] + snap1[2] + snap1[3] + snap1[4] + snap1[5] + snap1[6]
	total2 := snap2[0] + snap2[1] + snap2[2] + snap2[3] + snap2[4] + snap2[5] + snap2[6]
	idle1 := snap1[3]
	idle2 := snap2[3]

	deltaTotal := total2 - total1
	deltaIdle := idle2 - idle1

	if deltaTotal == 0 {
		return 0
	}
	return float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100
}

func readCPUStat() []uint64 {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 8 {
			return nil
		}
		vals := make([]uint64, 7)
		for i := 0; i < 7; i++ {
			v, err := strconv.ParseUint(fields[i+1], 10, 64)
			if err != nil {
				return nil
			}
			vals[i] = v
		}
		return vals
	}
	return nil
}

func readMemInfo() (total uint64, available uint64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total = val
		case "MemAvailable:":
			available = val
		}
		if total > 0 && available > 0 {
			break
		}
	}
	return total, available
}

// readJetsonGPU lee la carga GPU del Jetson Orin (0-1000 → 0-100%).
func readJetsonGPU() float64 {
	paths := []string{
		"/sys/devices/gpu.0/load",
		"/sys/devices/platform/17000000.gpu/load",
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		val, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		if err != nil {
			continue
		}
		return val / 10.0
	}
	return 0
}

// readJetsonGPUFreqs lee la frecuencia actual y máxima de la GPU en MHz
// desde el subsistema devfreq del kernel.
func readJetsonGPUFreqs() (curMHz, maxMHz int) {
	bases := []string{
		"/sys/devices/gpu.0/devfreq/17000000.gpu",
		"/sys/devices/platform/17000000.gpu/devfreq/17000000.gpu",
		"/sys/class/devfreq/17000000.gpu",
	}
	for _, base := range bases {
		curData, err1 := os.ReadFile(base + "/cur_freq")
		maxData, err2 := os.ReadFile(base + "/max_freq")
		if err1 != nil || err2 != nil {
			continue
		}
		cur, err1 := strconv.ParseInt(strings.TrimSpace(string(curData)), 10, 64)
		mx, err2 := strconv.ParseInt(strings.TrimSpace(string(maxData)), 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		// Los valores están en Hz → convertir a MHz
		return int(cur / 1_000_000), int(mx / 1_000_000)
	}
	return 0, 0
}

// readThermalZones lee las zonas térmicas de /sys/class/thermal.
func readThermalZones() []TempReading {
	zones, err := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")
	if err != nil || len(zones) == 0 {
		return nil
	}

	nameMap := map[string]string{
		"cpu-thermal":  "CPU",
		"gpu-thermal":  "GPU",
		"soc0":         "SoC 0",
		"soc1":         "SoC 1",
		"soc2":         "SoC 2",
		"tj-max":       "TJ Max",
		"CV0-therm":    "CV0",
		"CV1-therm":    "CV1",
		"CV2-therm":    "CV2",
		"CPU-therm":    "CPU",
		"GPU-therm":    "GPU",
		"PMIC-Die":     "PMIC",
		"BCPU-therm":   "CPU Big",
		"SCPU-therm":   "CPU Small",
		"MCPU-therm":   "CPU Mid",
		"AO-therm":     "Always-On",
		"Tboard_tegra": "Board",
		"Tdiode_tegra": "Diodo",
	}

	seen := map[string]bool{}
	var readings []TempReading

	for _, tempFile := range zones {
		data, err := os.ReadFile(tempFile)
		if err != nil {
			continue
		}
		milliC, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		if err != nil {
			continue
		}
		celsius := float64(milliC) / 1000.0
		if celsius < 10 || celsius > 120 {
			continue
		}

		dir := filepath.Dir(tempFile)
		zoneType := ""
		if raw, err := os.ReadFile(filepath.Join(dir, "type")); err == nil {
			zoneType = strings.TrimSpace(string(raw))
		} else {
			zoneType = fmt.Sprintf("Zone %s", filepath.Base(dir))
		}

		displayName := nameMap[zoneType]
		if displayName == "" {
			displayName = zoneType
		}
		if seen[displayName] {
			continue
		}
		seen[displayName] = true

		readings = append(readings, TempReading{Name: displayName, Celsius: celsius})
	}

	return readings
}
