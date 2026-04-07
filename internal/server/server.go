package server

import (
	"encoding/json"
	"github.com/stockyard-dev/stockyard-probe/internal/store"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type Server struct {
	db      *store.DB
	mux     *http.ServeMux
	limits  Limits
	dataDir string
	pCfg    map[string]json.RawMessage
}

func New(db *store.DB, limits Limits, dataDir string) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits, dataDir: dataDir}
	s.mux.HandleFunc("GET /api/bins", s.listBins)
	s.mux.HandleFunc("POST /api/bins", s.createBin)
	s.mux.HandleFunc("GET /api/bins/{id}", s.getBin)
	s.mux.HandleFunc("PUT /api/bins/{id}", s.updateBin)
	s.mux.HandleFunc("DELETE /api/bins/{id}", s.deleteBin)
	s.mux.HandleFunc("POST /api/bins/{id}/clear", s.clearRequests)
	s.mux.HandleFunc("GET /api/bins/{id}/requests", s.listRequests)
	s.mux.HandleFunc("GET /api/requests/{id}", s.getRequest)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)
	s.mux.HandleFunc("GET /api/tier", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"tier": s.limits.Tier, "upgrade_url": "https://stockyard.dev/probe/"})
	})
	s.loadPersonalConfig()
	s.mux.HandleFunc("GET /api/config", s.configHandler)
	s.mux.HandleFunc("GET /api/extras/{resource}", s.listExtras)
	s.mux.HandleFunc("GET /api/extras/{resource}/{id}", s.getExtras)
	s.mux.HandleFunc("PUT /api/extras/{resource}/{id}", s.putExtras)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Capture endpoint: any method to /b/{slug}
	if len(r.URL.Path) > 3 && r.URL.Path[:3] == "/b/" {
		s.capture(w, r)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui", http.StatusFound)
}

func (s *Server) capture(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Path[3:]
	bin := s.db.GetBinBySlug(slug)
	if bin == nil {
		writeErr(w, 404, "bin not found")
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	headers := map[string]string{}
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	req := &store.Request{BinID: bin.ID, Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Headers: headers, Body: string(body), IP: r.RemoteAddr, Size: len(body)}
	s.db.CaptureRequest(req)
	w.Header().Set("Content-Type", bin.ResponseType)
	w.WriteHeader(bin.ResponseCode)
	w.Write([]byte(bin.ResponseBody))
}

func (s *Server) listBins(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"bins": orEmpty(s.db.ListBins())})
}
func (s *Server) createBin(w http.ResponseWriter, r *http.Request) {
	var b store.Bin
	json.NewDecoder(r.Body).Decode(&b)
	if b.Name == "" || b.Slug == "" {
		writeErr(w, 400, "name and slug required")
		return
	}
	if err := s.db.CreateBin(&b); err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, s.db.GetBin(b.ID))
}
func (s *Server) getBin(w http.ResponseWriter, r *http.Request) {
	b := s.db.GetBin(r.PathValue("id"))
	if b == nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, b)
}
func (s *Server) updateBin(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ex := s.db.GetBin(id)
	if ex == nil {
		writeErr(w, 404, "not found")
		return
	}
	var b store.Bin
	json.NewDecoder(r.Body).Decode(&b)
	if b.Name == "" {
		b.Name = ex.Name
	}
	if b.ResponseCode <= 0 {
		b.ResponseCode = ex.ResponseCode
	}
	if b.ResponseType == "" {
		b.ResponseType = ex.ResponseType
	}
	s.db.UpdateBin(id, &b)
	writeJSON(w, 200, s.db.GetBin(id))
}
func (s *Server) deleteBin(w http.ResponseWriter, r *http.Request) {
	s.db.DeleteBin(r.PathValue("id"))
	writeJSON(w, 200, map[string]string{"deleted": "ok"})
}
func (s *Server) clearRequests(w http.ResponseWriter, r *http.Request) {
	s.db.ClearRequests(r.PathValue("id"))
	writeJSON(w, 200, map[string]string{"cleared": "ok"})
}
func (s *Server) listRequests(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	writeJSON(w, 200, map[string]any{"requests": orEmpty(s.db.ListRequests(r.PathValue("id"), limit))})
}
func (s *Server) getRequest(w http.ResponseWriter, r *http.Request) {
	req := s.db.GetRequest(r.PathValue("id"))
	if req == nil {
		writeErr(w, 404, "not found")
		return
	}
	writeJSON(w, 200, req)
}
func (s *Server) stats(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.db.Stats()) }
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	st := s.db.Stats()
	writeJSON(w, 200, map[string]any{"status": "ok", "service": "probe", "bins": st.Bins, "requests": st.Requests})
}
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
func init() { log.SetFlags(log.LstdFlags | log.Lshortfile) }

// ─── personalization (auto-added) ──────────────────────────────────

func (s *Server) loadPersonalConfig() {
	path := filepath.Join(s.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("%s: warning: could not parse config.json: %v", "probe", err)
		return
	}
	s.pCfg = cfg
	log.Printf("%s: loaded personalization from %s", "probe", path)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if s.pCfg == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pCfg)
}

func (s *Server) listExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	all := s.db.AllExtras(resource)
	out := make(map[string]json.RawMessage, len(all))
	for id, data := range all {
		out[id] = json.RawMessage(data)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) getExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	data := s.db.GetExtras(resource, id)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

func (s *Server) putExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read body"}`, 400)
		return
	}
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		http.Error(w, `{"error":"invalid json"}`, 400)
		return
	}
	if err := s.db.SetExtras(resource, id, string(body)); err != nil {
		http.Error(w, `{"error":"save failed"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":"saved"}`))
}
