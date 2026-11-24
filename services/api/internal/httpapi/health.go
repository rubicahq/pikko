package httpapi

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubicahq/pikko/services/api/internal/config"
	"github.com/rubicahq/pikko/services/api/internal/storage"
)

type healthDeps struct {
	DB      *pgxpool.Pool
	Cfg     *config.Config
	Storage storage.ObjectStore
}

func registerHealth(mux *http.ServeMux, d healthDeps) {
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"time": time.Now().UTC(),
		})
	})

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := d.DB.Ping(ctx); err != nil {
			writeError(w, r, http.StatusServiceUnavailable, "unhealthy", "Database unavailable.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"time": time.Now().UTC(),
			"env":  d.Cfg.AppEnv,
		})
	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		checks := map[string]string{}
		ready := true

		if err := d.DB.Ping(ctx); err != nil {
			checks["database"] = "fail"
			ready = false
		} else {
			checks["database"] = "ok"
			var one int
			if err := d.DB.QueryRow(ctx, `SELECT 1`).Scan(&one); err != nil {
				checks["database_query"] = "fail"
				ready = false
			} else {
				checks["database_query"] = "ok"
			}
		}

		if d.Cfg.StorageDriver == "local" {
			path := d.Cfg.StorageLocalPath
			if path == "" {
				path = "./.data/storage"
			}
			if err := os.MkdirAll(path, 0o755); err != nil {
				checks["storage"] = "fail"
				ready = false
			} else if st, err := os.Stat(path); err != nil || !st.IsDir() {
				checks["storage"] = "fail"
				ready = false
			} else {
				checks["storage"] = "ok"
			}
		} else if d.Storage != nil {
			checks["storage"] = "ok"
		} else {
			checks["storage"] = "unconfigured"
			ready = false
		}

		status := http.StatusOK
		if !ready {
			status = http.StatusServiceUnavailable
		}
		writeJSON(w, status, map[string]any{
			"ok":     ready,
			"time":   time.Now().UTC(),
			"checks": checks,
			"env":    d.Cfg.AppEnv,
		})
	})
}
