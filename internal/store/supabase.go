package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Forking-Around/SolarSense/internal/domain"
)

type Supabase struct {
	URL, PublishableKey string
	HTTP                *http.Client
}
type SavedReport struct {
	ID        string                  `json:"id"`
	CreatedAt time.Time               `json:"created_at"`
	Report    domain.AssessmentReport `json:"report"`
}
type KnowledgeResult struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Language    string `json:"language"`
	CountryCode string `json:"country_code"`
	State       string `json:"state"`
	Utility     string `json:"utility"`
	Topic       string `json:"topic"`
	SourceURL   string `json:"source_url"`
}

func (s Supabase) SaveReport(ctx context.Context, token, userID string, report domain.AssessmentReport) (string, error) {
	assessment := []map[string]any{{"user_id": userID, "status": "complete", "input": report.Input}}
	var created []struct {
		ID string `json:"id"`
	}
	if err := s.request(ctx, http.MethodPost, "/rest/v1/assessments", token, assessment, &created, "return=representation"); err != nil {
		return "", err
	}
	if len(created) != 1 {
		return "", fmt.Errorf("assessment insert returned no row")
	}
	payload := []map[string]any{{"assessment_id": created[0].ID, "user_id": userID, "report": report, "engine_version": report.EngineVersion, "solar_data_version": report.SolarSource.Version, "policy_version": report.Solar.PolicyVersion}}
	var rows []struct {
		ID string `json:"id"`
	}
	if err := s.request(ctx, http.MethodPost, "/rest/v1/reports", token, payload, &rows, "return=representation"); err != nil {
		return "", err
	}
	if len(rows) != 1 {
		return "", fmt.Errorf("report insert returned no row")
	}
	return rows[0].ID, nil
}
func (s Supabase) ListReports(ctx context.Context, token string) ([]SavedReport, error) {
	var out []SavedReport
	err := s.request(ctx, http.MethodGet, "/rest/v1/reports?select=id,created_at,report&order=created_at.desc", token, nil, &out, "")
	return out, err
}
func (s Supabase) DeleteReport(ctx context.Context, token, id string) error {
	return s.request(ctx, http.MethodDelete, "/rest/v1/reports?id=eq."+url.QueryEscape(id), token, nil, nil, "return=minimal")
}
func (s Supabase) SearchKnowledge(ctx context.Context, token, query string) ([]KnowledgeResult, error) {
	path := "/rest/v1/knowledge_documents?select=id,title,content,language,country_code,state,utility,topic,source_url&review_status=eq.reviewed&search_vector=fts." + url.QueryEscape(query) + "&limit=10"
	var out []KnowledgeResult
	err := s.request(ctx, http.MethodGet, path, token, nil, &out, "")
	return out, err
}
func (s Supabase) request(ctx context.Context, method, path, token string, in, out any, prefer string) error {
	var body *bytes.Reader
	if in == nil {
		body = bytes.NewReader(nil)
	} else {
		b, e := json.Marshal(in)
		if e != nil {
			return e
		}
		body = bytes.NewReader(b)
	}
	req, e := http.NewRequestWithContext(ctx, method, strings.TrimRight(s.URL, "/")+path, body)
	if e != nil {
		return e
	}
	req.Header.Set("apikey", s.PublishableKey)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	if prefer != "" {
		req.Header.Set("Prefer", prefer)
	}
	client := s.HTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, e := client.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Supabase data API returned %s", resp.Status)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
