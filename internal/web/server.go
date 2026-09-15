package web

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Forking-Around/SolarSense/internal/ai"
	"github.com/Forking-Around/SolarSense/internal/auth"
	"github.com/Forking-Around/SolarSense/internal/config"
	"github.com/Forking-Around/SolarSense/internal/domain"
	"github.com/Forking-Around/SolarSense/internal/engine"
	locationservice "github.com/Forking-Around/SolarSense/internal/location"
	"github.com/Forking-Around/SolarSense/internal/solar"
	"github.com/Forking-Around/SolarSense/internal/store"
)

//go:embed templates/*.html static/*
var assets embed.FS

type Server struct {
	cfg       config.Config
	log       *slog.Logger
	templates *template.Template
	nasa      solar.NASAClient
	auth      auth.Supabase
	extractor ai.Fallback
	geocoder  locationservice.Geocoder
	store     store.Supabase
}
type pageData struct {
	Lang            string
	T               map[string]string
	Report          *domain.AssessmentReport
	Error           string
	Form            map[string]string
	Months          []string
	Authenticated   bool
	CSRF            string
	SavedReports    []store.SavedReport
	Extraction      *domain.BillExtraction
	RoofObservation *domain.RoofPhotoObservation
}

func New(cfg config.Config, log *slog.Logger) (*Server, error) {
	funcs := template.FuncMap{"money": func(v float64) string { return fmt.Sprintf("₹%.0f", v) }, "num": func(v float64) string { return fmt.Sprintf("%.1f", v) }, "bar": func(v, max float64) int {
		if max <= 0 {
			return 0
		}
		return int(math.Round(v / max * 100))
	}, "join": strings.Join, "contains": strings.Contains, "reportToken": func(v domain.AssessmentReport) string { return signReport(v, cfg.ReportSigningKey) }}
	t, err := template.New("root").Funcs(funcs).ParseFS(assets, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, log: log, templates: t, nasa: solar.NASAClient{Endpoint: cfg.NASAPowerURL}, auth: auth.Supabase{URL: cfg.SupabaseURL, PublishableKey: cfg.SupabasePublishableKey, PublicURL: cfg.PublicURL}, extractor: ai.Fallback{Primary: ai.Gemini{APIKey: cfg.GeminiAPIKey, Model: cfg.GeminiModel, BaseURL: cfg.GeminiAPIBaseURL}, Secondary: ai.Groq{APIKey: cfg.GroqAPIKey, Model: cfg.GroqModel, Endpoint: cfg.GroqAPIURL}}, geocoder: locationservice.Geocoder{Endpoint: cfg.GeocodingAPIURL, APIKey: cfg.GoogleMapsKey}, store: store.Supabase{URL: cfg.SupabaseURL, PublishableKey: cfg.SupabasePublishableKey}}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.Handle("GET /static/", http.FileServer(http.FS(assets)))
	mux.HandleFunc("GET /", s.home)
	mux.HandleFunc("POST /assessment", s.assess)
	mux.HandleFunc("GET /auth/google", s.authGoogle)
	mux.HandleFunc("GET /auth/callback", s.authCallback)
	mux.HandleFunc("POST /uploads", s.uploadBill)
	mux.HandleFunc("POST /roof-observations", s.uploadRoof)
	mux.HandleFunc("POST /reports", s.saveReport)
	mux.HandleFunc("GET /reports", s.listReports)
	mux.HandleFunc("POST /reports/{id}/delete", s.deleteReport)
	mux.HandleFunc("GET /knowledge", s.searchKnowledge)
	return securityHeaders(s.requestLog(mux, s.log))
}

func (s *Server) render(w http.ResponseWriter, name string, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		s.log.Error("render", "error", err)
	}
}
func lang(r *http.Request) string {
	l := r.URL.Query().Get("lang")
	if l != "te" && l != "hi" {
		return "en"
	}
	return l
}
func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	l := lang(r)
	_, err := s.auth.User(r.Context(), r)
	token := ensureCSRF(w, r, strings.HasPrefix(s.cfg.PublicURL, "https://"))
	s.render(w, "index.html", pageData{Lang: l, T: language(l), Months: monthNames(), Authenticated: err == nil, CSRF: token})
}

func f(r *http.Request, key string, fallback float64) float64 {
	v, e := strconv.ParseFloat(r.FormValue(key), 64)
	if e != nil {
		return fallback
	}
	return v
}
func sun(v string) float64 {
	switch v {
	case "full":
		return 1
	case "partial":
		return .65
	case "none":
		return .2
	}
	return .65
}
func (s *Server) assess(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", 400)
		return
	}
	if !validCSRF(r) {
		http.Error(w, "This form expired. Refresh the page and try again.", http.StatusForbidden)
		return
	}
	lat, latOK := locationservice.ParseCoordinate(r.FormValue("latitude"))
	lon, lonOK := locationservice.ParseCoordinate(r.FormValue("longitude"))
	place, postal := r.FormValue("location"), r.FormValue("pincode")
	if (!latOK || !lonOK) && s.geocoder.Enabled() {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		if found, err := s.geocoder.Search(ctx, strings.TrimSpace(place+" "+postal+" India")); err == nil {
			lat, lon, latOK, lonOK, place = found.Latitude, found.Longitude, true, true, found.Name
			if postal == "" {
				postal = found.PostalCode
			}
		} else {
			s.log.Warn("geocoding fallback", "error", err)
		}
	}
	resource := solar.IndiaScreeningResource()
	if latOK {
		resource.Latitude = lat
	}
	if s.cfg.LiveSolar && latOK && lonOK {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		if live, err := s.nasa.Resource(ctx, lat, lon); err == nil {
			resource = live
		} else {
			s.log.Warn("solar data fallback", "error", err)
		}
	}
	roof := domain.RoofProfile{Type: r.FormValue("roof_type"), Ownership: r.FormValue("ownership"), Zones: []domain.RoofZone{{Name: "Main roof", GrossSqFt: f(r, "roof_area", 500), ObstaclePct: f(r, "obstacles", 15), Orientation: r.FormValue("orientation"), TiltDegrees: f(r, "tilt", 10), Sun: domain.SunlightWindow{Morning: sun(r.FormValue("morning")), Midday: sun(r.FormValue("midday")), Evening: sun(r.FormValue("evening"))}}}}
	in := domain.AssessmentInput{Location: domain.LocationContext{Name: place, Country: "India", PostalCode: postal, Utility: r.FormValue("utility"), Latitude: lat, Longitude: lon}, MonthlyBillINR: f(r, "bill", 2500), MonthlyUnitsKWh: f(r, "units", 0), DaytimeUsePct: f(r, "day_use", 45), Roof: roof, Outages: domain.OutageProfile{FrequencyPerMonth: f(r, "outages", 2), HoursPerOutage: f(r, "outage_hours", 1), EssentialWatts: f(r, "essential_watts", 500), DesiredHours: f(r, "backup_hours", 4)}, GridRateINR: f(r, "grid_rate", 8), ExportRateINR: f(r, "export_rate", 2.5), FixedChargeINR: f(r, "fixed_charge", 150), SanctionedLoadKW: f(r, "sanctioned_load", 0)}
	if err := validateInput(in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	report := engine.Assess(in, resource)
	report.Solar.PolicyVersion, report.Solar.PolicySourceURL = s.cfg.IndiaPolicyVersion, s.cfg.IndiaPolicySourceURL
	if resource.Version == "screening-v1" || strings.TrimSpace(in.Location.Utility) == "" || report.Solar.PolicySourceURL == "" {
		report.Solar.Confidence = "preliminary"
		report.Solar.Verdict = "Preliminary — " + report.Solar.Verdict
		report.Solar.Warnings = append(report.Solar.Warnings, "A verified utility tariff/export policy and live local solar data are required for a firm verdict.")
	}
	l := r.FormValue("lang")
	if l == "" {
		l = "en"
	}
	_, authErr := s.auth.User(r.Context(), r)
	data := pageData{Lang: l, T: language(l), Report: &report, Months: monthNames(), CSRF: r.FormValue("csrf_token"), Authenticated: authErr == nil}
	if r.Header.Get("HX-Request") == "true" {
		s.render(w, "result.html", data)
		return
	}
	s.render(w, "index.html", data)
}

func validateInput(in domain.AssessmentInput) error {
	if strings.TrimSpace(in.Location.Name) == "" {
		return fmt.Errorf("enter the installation city, town, or village")
	}
	if in.MonthlyBillINR < 0 || in.MonthlyUnitsKWh < 0 {
		return fmt.Errorf("bill and consumption cannot be negative")
	}
	if in.GridRateINR <= 0 || in.ExportRateINR < 0 {
		return fmt.Errorf("enter valid grid and export rates")
	}
	if in.DaytimeUsePct < 0 || in.DaytimeUsePct > 100 {
		return fmt.Errorf("daytime usage must be between 0 and 100 percent")
	}
	if len(in.Roof.Zones) == 0 || in.Roof.Zones[0].GrossSqFt < 30 {
		return fmt.Errorf("enter at least 30 sq.ft of gross roof area")
	}
	if in.Roof.Zones[0].ObstaclePct < 0 || in.Roof.Zones[0].ObstaclePct > 85 {
		return fmt.Errorf("blocked roof area must be between 0 and 85 percent")
	}
	if in.Outages.FrequencyPerMonth < 0 || in.Outages.HoursPerOutage < 0 || in.Outages.EssentialWatts <= 0 || in.Outages.DesiredHours <= 0 {
		return fmt.Errorf("enter valid power-cut and backup details")
	}
	return nil
}

func (s *Server) authGoogle(w http.ResponseWriter, r *http.Request) {
	if !s.auth.Enabled() {
		http.Error(w, "Google sign-in needs SUPABASE_URL and SUPABASE_PUBLISHABLE_KEY.", http.StatusServiceUnavailable)
		return
	}
	if err := s.auth.Begin(w, r); err != nil {
		http.Error(w, "Could not start sign-in.", 500)
	}
}
func (s *Server) authCallback(w http.ResponseWriter, r *http.Request) {
	if err := s.auth.Callback(r.Context(), w, r); err != nil {
		s.log.Warn("auth callback", "error", err)
		http.Error(w, "Sign-in could not be completed.", http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func (s *Server) uploadBill(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Upload must be 10 MB or smaller.", http.StatusBadRequest)
		return
	}
	if !validCSRF(r) {
		http.Error(w, "This form expired. Refresh the page and try again.", http.StatusForbidden)
		return
	}
	if _, err := s.auth.User(r.Context(), r); err != nil {
		http.Error(w, "Sign in before uploading a bill.", http.StatusUnauthorized)
		return
	}
	file, _, err := r.FormFile("bill_file")
	if err != nil {
		http.Error(w, "Choose a bill image or PDF.", 400)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 10<<20))
	if err != nil {
		http.Error(w, "Could not read upload.", 400)
		return
	}
	detected := http.DetectContentType(data[:min(len(data), 512)])
	mime := detected
	if detected != "application/pdf" && detected != "image/jpeg" && detected != "image/png" {
		http.Error(w, "The file contents are not a JPG, PNG, or PDF.", http.StatusBadRequest)
		return
	}
	result, err := s.extractor.ExtractBill(r.Context(), ai.Document{MIMEType: mime, Data: data})
	if err != nil {
		s.log.Warn("bill extraction", "error", err)
		http.Error(w, "We could not read this bill. Enter the details manually.", 422)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		s.render(w, "bill-extraction.html", pageData{Extraction: &result})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) uploadRoof(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Upload must be 10 MB or smaller.", http.StatusBadRequest)
		return
	}
	if !validCSRF(r) {
		http.Error(w, "This form expired. Refresh and try again.", http.StatusForbidden)
		return
	}
	if _, err := s.auth.User(r.Context(), r); err != nil {
		http.Error(w, "Sign in before uploading a roof photo.", http.StatusUnauthorized)
		return
	}
	file, _, err := r.FormFile("roof_file")
	if err != nil {
		http.Error(w, "Choose a roof photo.", http.StatusBadRequest)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 10<<20))
	if err != nil {
		http.Error(w, "Could not read upload.", http.StatusBadRequest)
		return
	}
	mime := http.DetectContentType(data[:min(len(data), 512)])
	if mime != "image/jpeg" && mime != "image/png" {
		http.Error(w, "Use a JPG or PNG roof photo.", http.StatusBadRequest)
		return
	}
	result, err := s.extractor.ExtractRoof(r.Context(), ai.Document{MIMEType: mime, Data: data})
	if err != nil {
		s.log.Warn("roof observation", "error", err)
		http.Error(w, "We could not inspect this roof photo. Continue with the manual roof answers.", http.StatusUnprocessableEntity)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		s.render(w, "roof-observation.html", pageData{RoofObservation: &result})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) saveReport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || !validCSRF(r) {
		http.Error(w, "This form expired. Refresh and try again.", http.StatusForbidden)
		return
	}
	user, err := s.auth.User(r.Context(), r)
	if err != nil {
		http.Error(w, "Sign in before saving a report.", http.StatusUnauthorized)
		return
	}
	report, err := verifyReport(r.FormValue("report"), s.cfg.ReportSigningKey)
	if err != nil || report.EngineVersion != engine.Version {
		http.Error(w, "This report cannot be saved; calculate it again.", http.StatusBadRequest)
		return
	}
	if _, err := s.store.SaveReport(r.Context(), mustToken(r), user.ID, report); err != nil {
		s.log.Error("save report", "error", err)
		http.Error(w, "Report could not be saved.", http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/reports", http.StatusSeeOther)
}

func (s *Server) listReports(w http.ResponseWriter, r *http.Request) {
	if _, err := s.auth.User(r.Context(), r); err != nil {
		http.Redirect(w, r, "/auth/google", http.StatusSeeOther)
		return
	}
	reports, err := s.store.ListReports(r.Context(), mustToken(r))
	if err != nil {
		http.Error(w, "Reports could not be loaded.", http.StatusBadGateway)
		return
	}
	csrf := ensureCSRF(w, r, strings.HasPrefix(s.cfg.PublicURL, "https://"))
	s.render(w, "reports.html", pageData{Lang: "en", T: language("en"), SavedReports: reports, CSRF: csrf})
}

func (s *Server) deleteReport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || !validCSRF(r) {
		http.Error(w, "This form expired.", http.StatusForbidden)
		return
	}
	if _, err := s.auth.User(r.Context(), r); err != nil {
		http.Error(w, "Sign in required.", http.StatusUnauthorized)
		return
	}
	if err := s.store.DeleteReport(r.Context(), mustToken(r), r.PathValue("id")); err != nil {
		http.Error(w, "Report could not be deleted.", http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/reports", http.StatusSeeOther)
}

func (s *Server) searchKnowledge(w http.ResponseWriter, r *http.Request) {
	if _, err := s.auth.User(r.Context(), r); err != nil {
		http.Error(w, "Sign in required.", http.StatusUnauthorized)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		http.Error(w, "Search query is required.", http.StatusBadRequest)
		return
	}
	rows, err := s.store.SearchKnowledge(r.Context(), mustToken(r), q)
	if err != nil {
		http.Error(w, "Knowledge search is unavailable.", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rows)
}

func mustToken(r *http.Request) string { v, _ := auth.AccessToken(r); return v }

func signReport(report domain.AssessmentReport, key string) string {
	body, _ := json.Marshal(report)
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write(body)
	return base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func verifyReport(token, key string) (domain.AssessmentReport, error) {
	var report domain.AssessmentReport
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return report, fmt.Errorf("invalid signed report")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return report, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return report, err
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write(body)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return report, fmt.Errorf("report signature mismatch")
	}
	if err := json.Unmarshal(body, &report); err != nil {
		return report, err
	}
	return report, nil
}

func ensureCSRF(w http.ResponseWriter, r *http.Request, secure bool) string {
	if c, err := r.Cookie("solarsense_csrf"); err == nil && len(c.Value) >= 32 {
		return c.Value
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("secure random unavailable")
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{Name: "solarsense_csrf", Value: token, Path: "/", Secure: secure, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 86400})
	return token
}
func validCSRF(r *http.Request) bool {
	c, err := r.Cookie("solarsense_csrf")
	if err != nil {
		return false
	}
	v := r.FormValue("csrf_token")
	if len(v) != len(c.Value) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(v), []byte(c.Value)) == 1
}
func monthNames() []string {
	return []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(self)")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}
func (s *Server) requestLog(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}
