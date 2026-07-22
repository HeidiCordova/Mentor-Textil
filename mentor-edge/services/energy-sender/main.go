package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── HTTP client compartido ─────────────────────────────────────────────────
// Un único cliente reutilizado por todos los workers: mantiene keep-alive y
// evita crear un socket nuevo por cada lote enviado.
var httpClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 8,
		IdleConnTimeout:     90 * time.Second,
	},
}

// ─── Dominio ─────────────────────────────────────────────────────────────────

type snapshot struct {
	DeviceID  string   `json:"device_id"`
	MeterID   string   `json:"meter_id"`
	Timestamp int64    `json:"timestamp"`
	IntervalS int      `json:"interval_s"`
	Head      []string `json:"head"`
	Data      []string `json:"data"`
	Ubicacion string   `json:"ubicacion,omitempty"`
}

type pendingCommand struct {
	ID      int64           `json:"id"`
	Command string          `json:"command"`
	Payload json.RawMessage `json:"payload"`
}

// ─── Config desde DB ─────────────────────────────────────────────────────────

type runtimeConfig struct {
	mu            sync.RWMutex
	deviceID      string
	cloudURL      string
	energyAPIKey  string
	sendIntervalS int
	batchSize     int
	reloadS       int
	maxWorkers    int
	writeAPIURL   string
	commandPollS  int
}

func (rc *runtimeConfig) load(ctx context.Context, db *pgxpool.Pool) error {
	rows, err := db.Query(ctx, `SELECT key, value FROM energy.config`)
	if err != nil {
		return err
	}
	defer rows.Close()

	m := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return err
		}
		m[k] = v
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()
	if v, ok := m["device_id"]; ok && v != "" {
		rc.deviceID = v
	}
	if v, ok := m["cloud_url"]; ok && v != "" {
		rc.cloudURL = v
	}
	if v, ok := m["energy_api_key"]; ok {
		rc.energyAPIKey = v
	}
	if v, ok := m["send_interval_s"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rc.sendIntervalS = n
		}
	}
	if v, ok := m["batch_size"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rc.batchSize = n
		}
	}
	if v, ok := m["config_reload_s"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rc.reloadS = n
		}
	}
	if v, ok := m["max_workers"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rc.maxWorkers = n
		}
	}
	if v, ok := m["write_api_url"]; ok && v != "" {
		rc.writeAPIURL = v
	}
	if v, ok := m["command_poll_s"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rc.commandPollS = n
		}
	}
	return nil
}

func (rc *runtimeConfig) get() (deviceID, cloudURL, apiKey string, sendS, batchSz int) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.deviceID, rc.cloudURL, rc.energyAPIKey, rc.sendIntervalS, rc.batchSize
}

func (rc *runtimeConfig) getMaxWorkers() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if rc.maxWorkers <= 0 {
		return 4
	}
	return rc.maxWorkers
}

func (rc *runtimeConfig) getWriteAPIURL() string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if rc.writeAPIURL == "" {
		return "http://localhost:8088/write-api"
	}
	return rc.writeAPIURL
}

func (rc *runtimeConfig) getCommandPollS() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if rc.commandPollS <= 0 {
		return 3
	}
	return rc.commandPollS
}

// ─── Main ─────────────────────────────────────────────────────────────────────

func main() {
	// Unicas vars de infra inmutable — no cambian sin recrear el contenedor
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = "postgresql://mentor:mentor@postgres:5432/mentor_energy"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, postgresURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	cfg := &runtimeConfig{
		deviceID:      "rpi-energy-01",
		cloudURL:      "https://mentormonitor-ai.com",
		sendIntervalS: 30,
		batchSize:     50,
		reloadS:       60,
		maxWorkers:    4,
		writeAPIURL:   "http://localhost:8088/write-api",
		commandPollS:  3,
	}

	if err := cfg.load(ctx, db); err != nil {
		log.Printf("config load (DB puede estar inicializando): %v", err)
	}

	deviceID, cloudURL, _, _, _ := cfg.get()
	log.Printf("energy-sender started device=%s cloud=%s", deviceID, cloudURL)

	// Recarga de config en caliente desde DB
	go func() {
		for {
			cfg.mu.RLock()
			reloadS := cfg.reloadS
			cfg.mu.RUnlock()
			time.Sleep(time.Duration(reloadS) * time.Second)

			if err := cfg.load(ctx, db); err != nil {
				log.Printf("config reload: %v", err)
			}
		}
	}()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		devID, curl, _, _, _ := cfg.get()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"service":"energy-sender","status":"ok","device":"%s","cloud":"%s"}`, devID, curl)
	})
	http.HandleFunc("/config/reload", func(w http.ResponseWriter, r *http.Request) {
		if err := cfg.load(r.Context(), db); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"reloaded"}`)
	})
	registerWebHandlers(db, cfg)
	log.Printf("http: iniciando servidor en :%s", port)
	go func() {
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Printf("http: error fatal: %v", err)
		}
	}()

	go func() {
		for {
			poll := cfg.getCommandPollS()
			if poll <= 0 {
				poll = 3
			}
			processPendingCommands(ctx, db, cfg)
			time.Sleep(time.Duration(poll) * time.Second)
		}
	}()

	// Loop de envio con intervalo leido desde DB
	var ticker *time.Ticker
	currentInterval := 0
	startupDrainDone := false

	for {
		_, _, apiKey, sendS, _ := cfg.get()

		if apiKey == "" {
			log.Printf("energy_api_key no configurado en energy.config — esperando...")
			time.Sleep(10 * time.Second)
			continue
		}

		// Primera vez con API key válida: drenar el backlog acumulado
		// durante la desconexión antes de esperar el primer tick.
		if !startupDrainDone {
			log.Printf("[energy-sender] drenando backlog pendiente al arrancar...")
			drainAll(ctx, db, cfg)
			startupDrainDone = true
		}

		if sendS != currentInterval {
			if ticker != nil {
				ticker.Stop()
			}
			ticker = time.NewTicker(time.Duration(sendS) * time.Second)
			currentInterval = sendS
			log.Printf("send_interval actualizado a %ds", sendS)
		}

		select {
		case <-ticker.C:
			drainAll(ctx, db, cfg)
		}
	}
}

func processPendingCommands(ctx context.Context, db *pgxpool.Pool, cfg *runtimeConfig) {
	deviceID, cloudURL, apiKey, _, _ := cfg.get()
	if deviceID == "" || cloudURL == "" || apiKey == "" {
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(cloudURL, "/")+"/api/v1/edge/pending-commands", nil)
	if err != nil {
		return
	}
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("X-Device-ID", deviceID)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("pending-commands fetch: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}

	var cmds []pendingCommand
	if err := json.NewDecoder(resp.Body).Decode(&cmds); err != nil || len(cmds) == 0 {
		return
	}

	applied := make([]int64, 0, len(cmds))
	for _, cmd := range cmds {
		if err := applyPendingCommand(ctx, db, cfg, cmd); err != nil {
			log.Printf("pending command %d (%s) failed: %v", cmd.ID, cmd.Command, err)
			continue
		}
		applied = append(applied, cmd.ID)
	}
	if len(applied) == 0 {
		return
	}

	body, _ := json.Marshal(map[string]interface{}{"ids": applied})
	ackReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cloudURL, "/")+"/api/v1/edge/pending-commands/ack", bytes.NewReader(body))
	if err != nil {
		return
	}
	ackReq.Header.Set("Content-Type", "application/json")
	ackReq.Header.Set("X-API-Key", apiKey)
	ackReq.Header.Set("X-Device-ID", deviceID)
	ackResp, err := httpClient.Do(ackReq)
	if err != nil {
		log.Printf("pending-commands ack: %v", err)
		return
	}
	_ = ackResp.Body.Close()
	if ackResp.StatusCode == http.StatusOK {
		log.Printf("acked %d pending commands", len(applied))
	}
}

func applyPendingCommand(ctx context.Context, db *pgxpool.Pool, cfg *runtimeConfig, cmd pendingCommand) error {
	switch cmd.Command {
	case "energy_config_upsert":
		return applyEnergyConfigUpsert(ctx, db, cfg, cmd.Payload)
	case "energy_meter_upsert":
		return applyEnergyMeterUpsert(ctx, db, cmd.Payload)
	case "energy_meter_delete":
		return applyEnergyMeterDelete(ctx, db, cmd.Payload)
	case "energy_mc60_command":
		return applyEnergyMC60Command(ctx, cfg, cmd.Payload)
	default:
		return nil
	}
}

func applyEnergyConfigUpsert(ctx context.Context, db *pgxpool.Pool, cfg *runtimeConfig, payload json.RawMessage) error {
	var req struct {
		Values map[string]string `json:"values"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return err
	}
	if len(req.Values) == 0 {
		return nil
	}
	allowed := map[string]bool{
		"device_id": true, "cloud_url": true, "energy_api_key": true,
		"send_interval_s": true, "batch_size": true, "config_reload_s": true,
		"write_api_url": true, "command_poll_s": true,
	}
	for k, v := range req.Values {
		if !allowed[k] {
			continue
		}
		if _, err := db.Exec(ctx,
			`INSERT INTO energy.config (key, value, updated_at) VALUES ($1, $2, NOW())
			 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			k, v,
		); err != nil {
			return err
		}
	}
	return cfg.load(ctx, db)
}

func applyEnergyMeterUpsert(ctx context.Context, db *pgxpool.Pool, payload json.RawMessage) error {
	var req struct {
		MeterID   string  `json:"meter_id"`
		UnitID    int     `json:"unit_id"`
		Ubicacion *string `json:"ubicacion"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return err
	}
	if req.MeterID == "" || req.UnitID < 1 || req.UnitID > 247 {
		return fmt.Errorf("meter_id/unit_id invalid")
	}
	_, err := db.Exec(ctx,
		`INSERT INTO energy.meters (meter_id, unit_id, ubicacion, active)
		 VALUES ($1, $2, $3, TRUE)
		 ON CONFLICT (meter_id)
		 DO UPDATE SET unit_id = EXCLUDED.unit_id,
		               ubicacion = COALESCE(EXCLUDED.ubicacion, energy.meters.ubicacion),
		               active = TRUE`,
		req.MeterID, req.UnitID, req.Ubicacion,
	)
	return err
}

func applyEnergyMeterDelete(ctx context.Context, db *pgxpool.Pool, payload json.RawMessage) error {
	var req struct {
		MeterID string `json:"meter_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return err
	}
	if req.MeterID == "" {
		return fmt.Errorf("meter_id required")
	}
	_, err := db.Exec(ctx, `UPDATE energy.meters SET active = FALSE WHERE meter_id = $1`, req.MeterID)
	return err
}

func applyEnergyMC60Command(ctx context.Context, cfg *runtimeConfig, payload json.RawMessage) error {
	var req struct {
		UnitID  int                    `json:"unit_id"`
		Command string                 `json:"command"`
		Params  map[string]interface{} `json:"params"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return err
	}
	if req.Params == nil {
		req.Params = map[string]interface{}{}
	}
	base := strings.TrimRight(cfg.getWriteAPIURL(), "/")
	if base == "" {
		base = "http://localhost:8088/write-api"
	}

	var endpoint string
	if req.Command == "scan" {
		endpoint = base + "/scan"
	} else {
		if req.UnitID < 1 || req.UnitID > 247 {
			return fmt.Errorf("unit_id invalid")
		}
		endpoint = fmt.Sprintf("%s/meter/%d/%s", base, req.UnitID, req.Command)
	}
	body, _ := json.Marshal(req.Params)
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(hreq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mc60 command failed status=%d", resp.StatusCode)
	}
	return nil
}

// ─── Envio al cloud ───────────────────────────────────────────────────────────

// getActiveMeterIDs retorna los meter_id distintos con snapshots pendientes de sync.
func getActiveMeterIDs(ctx context.Context, db *pgxpool.Pool) []string {
	rows, err := db.Query(ctx, `SELECT DISTINCT meter_id FROM energy.snapshots WHERE NOT synced ORDER BY meter_id`)
	if err != nil {
		log.Printf("getActiveMeterIDs: %v", err)
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// drainAll lanza un worker concurrente por cada meter_id con snapshots pendientes,
// limitado a cfg.maxWorkers goroutines simultáneas. Con un solo medidor el
// comportamiento es idéntico al anterior; con N medidores se envían en paralelo.
func drainAll(ctx context.Context, db *pgxpool.Pool, cfg *runtimeConfig) {
	meterIDs := getActiveMeterIDs(ctx, db)
	if len(meterIDs) == 0 {
		return
	}

	maxW := cfg.getMaxWorkers()
	sem := make(chan struct{}, maxW)
	var wg sync.WaitGroup

	for _, mid := range meterIDs {
		wg.Add(1)
		sem <- struct{}{}
		go func(meterID string) {
			defer wg.Done()
			defer func() { <-sem }()
			lotes := 0
			for sendBatchForMeter(ctx, db, cfg, meterID) > 0 {
				lotes++
			}
			if lotes > 1 {
				log.Printf("[drain] meter=%s vaciado en %d lotes", meterID, lotes)
			}
		}(mid)
	}
	wg.Wait()
}

// sendBatchForMeter envía un lote de snapshots de un meter_id específico al cloud.
// Retorna la cantidad de registros enviados (0 si no hay pendientes o hay error).
func sendBatchForMeter(ctx context.Context, db *pgxpool.Pool, cfg *runtimeConfig, meterID string) int {
	deviceID, cloudURL, apiKey, _, batchSize := cfg.get()

	if apiKey == "" {
		return 0
	}

	// Atributos por medidor: ubicacion, nombre en la nube y token exclusivo.
	var meterUbicacion, meterCloudName, meterCloudToken string
	db.QueryRow(ctx,
		`SELECT COALESCE(ubicacion,''), COALESCE(cloud_name,''), COALESCE(cloud_token,'')
		   FROM energy.meters WHERE meter_id = $1`, meterID).
		Scan(&meterUbicacion, &meterCloudName, &meterCloudToken)

	// El token por medidor tiene prioridad sobre la clave global.
	effectiveAPIKey := apiKey
	if meterCloudToken != "" {
		effectiveAPIKey = meterCloudToken
	}
	// El nombre en la nube por medidor tiene prioridad sobre meter_id.
	effectiveMeterID := meterID
	if meterCloudName != "" {
		effectiveMeterID = meterCloudName
	}

	type row struct {
		id        int64
		meterID   string
		hora      time.Time
		intervalS int
		head      []string
		data      []string
	}

	rows, err := db.Query(ctx, `
SELECT id, meter_id, hora, interval_s, head, data
  FROM energy.snapshots
 WHERE NOT synced AND meter_id = $2
 ORDER BY hora ASC
 LIMIT $1`, batchSize, meterID)
	if err != nil {
		log.Printf("query meter=%s: %v", meterID, err)
		return 0
	}
	defer rows.Close()

	var batch []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.meterID, &r.hora, &r.intervalS, &r.head, &r.data); err != nil {
			log.Printf("scan meter=%s: %v", meterID, err)
			continue
		}
		batch = append(batch, r)
	}
	rows.Close()

	if len(batch) == 0 {
		return 0
	}

	payload := make([]snapshot, 0, len(batch))
	var ids []int64
	for _, r := range batch {
		payload = append(payload, snapshot{
			DeviceID:  deviceID,
			MeterID:   effectiveMeterID,
			Timestamp: r.hora.Unix(),
			IntervalS: r.intervalS,
			Head:      r.head,
			Data:      r.data,
			Ubicacion: meterUbicacion,
		})
		ids = append(ids, r.id)
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		cloudURL+"/api/v1/energy/snapshots", bytes.NewReader(body))
	if err != nil {
		log.Printf("build request meter=%s: %v", meterID, err)
		return 0
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", effectiveAPIKey)
	req.Header.Set("X-Device-ID", deviceID)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("send meter=%s: %v", meterID, err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("cloud returned %d for meter=%s batch=%d", resp.StatusCode, meterID, len(ids))
		return 0
	}

	if _, err := db.Exec(ctx, `UPDATE energy.snapshots SET synced = TRUE WHERE id = ANY($1)`, ids); err != nil {
		log.Printf("mark synced meter=%s: %v", meterID, err)
		return 0
	}

	log.Printf("synced %d snapshots meter=%s → cloud", len(ids), meterID)
	return len(ids)
}
