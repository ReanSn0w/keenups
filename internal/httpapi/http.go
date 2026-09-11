package httpapi

import (
	"encoding/json"
	"html/template"
	"net/http"
	"sort"
	"time"

	"github.com/reansnow/keenups/internal/state"
)

type row struct {
	Name  string
	Value string
}

type pageData struct {
	Version string
	State   state.Snapshot
	Rows    []row
}

var page = template.Must(template.New("status").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta http-equiv="refresh" content="5"><title>KeenUPS</title>
<style>body{font:16px system-ui;margin:2rem;max-width:60rem;color:#172033}table{border-collapse:collapse;width:100%}td,th{padding:.45rem .7rem;border-bottom:1px solid #dde2ea;text-align:left}.ok{color:#08783e}.bad{color:#b42318}code{background:#f1f4f8;padding:.1rem .3rem}</style></head>
<body><h1>KeenUPS</h1><p>Driver: {{if .State.Ready}}<strong class="ok">ready</strong>{{else}}<strong class="bad">not ready</strong>{{end}} · Version <code>{{.Version}}</code></p>
<table><thead><tr><th>Variable</th><th>Value</th></tr></thead><tbody>{{range .Rows}}<tr><td><code>{{.Name}}</code></td><td>{{.Value}}</td></tr>{{end}}</tbody></table></body></html>`))

func Handler(store *state.Store, version string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		snap := store.Snapshot()
		if !snap.Ready {
			snap.Variables = map[string]string{}
		}
		keys := make([]string, 0, len(snap.Variables))
		for key := range snap.Variables {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		rows := make([]row, 0, len(keys))
		for _, key := range keys {
			rows = append(rows, row{key, snap.Variables[key]})
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, pageData{Version: version, State: snap, Rows: rows})
	})
	mux.HandleFunc("GET /api/v1/status", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		snap := store.Snapshot()
		if !snap.Ready {
			snap.Variables = map[string]string{}
		}
		_ = json.NewEncoder(w).Encode(snap)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		snap := store.Snapshot()
		w.Header().Set("Content-Type", "application/json")
		if !snap.Ready {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ready":       snap.Ready,
			"connected":   snap.Connected,
			"last_update": snap.LastUpdate.Format(time.RFC3339),
		})
	})
	return mux
}
