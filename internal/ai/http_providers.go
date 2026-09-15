package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Forking-Around/SolarSense/internal/domain"
)

const billPrompt = `Read this electricity bill. Return JSON only with utility, billing_period, tariff_category, units_kwh, amount, sanctioned_load_kw, and confidence. confidence is an object with a 0..1 number for each extracted field. Use zero and low confidence when unreadable. Never infer a value that is not visible.`
const roofPrompt = `Inspect this rooftop photo. Return JSON only with roof_type, visible_obstacles (array), obstacle_estimate_pct, partial_shade_visible, has_scale_reference, and confidence. Do not infer dimensions, direction, or all-day sunlight from one photo. obstacle_estimate_pct is only the visible fraction and must have low confidence without a scale reference.`

type Gemini struct {
	APIKey, Model, BaseURL string
	HTTP                   *http.Client
}

func (g Gemini) Name() string { return "gemini" }
func (g Gemini) ExtractBill(ctx context.Context, d Document) (domain.BillExtraction, error) {
	if g.APIKey == "" {
		return domain.BillExtraction{}, ErrUnavailable
	}
	body := map[string]any{
		"contents": []any{map[string]any{"role": "user", "parts": []any{
			map[string]any{"text": billPrompt},
			map[string]any{"inline_data": map[string]any{"mime_type": d.MIMEType, "data": base64.StdEncoding.EncodeToString(d.Data)}},
		}}},
		"generationConfig": map[string]any{"responseMimeType": "application/json"},
	}
	if g.BaseURL == "" {
		return domain.BillExtraction{}, fmt.Errorf("gemini endpoint is not configured")
	}
	endpoint := fmt.Sprintf("%s/%s:generateContent", strings.TrimRight(g.BaseURL, "/"), g.Model)
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := postJSON(ctx, g.client(), endpoint, map[string]string{"x-goog-api-key": g.APIKey}, body, &result); err != nil {
		return domain.BillExtraction{}, err
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return domain.BillExtraction{}, fmt.Errorf("gemini returned no extraction")
	}
	return decodeBill(result.Candidates[0].Content.Parts[0].Text)
}
func (g Gemini) ExtractRoof(ctx context.Context, d Document) (domain.RoofPhotoObservation, error) {
	if g.APIKey == "" || g.BaseURL == "" { return domain.RoofPhotoObservation{}, ErrUnavailable }
	body := map[string]any{"contents": []any{map[string]any{"role":"user","parts":[]any{map[string]any{"text":roofPrompt},map[string]any{"inline_data":map[string]any{"mime_type":d.MIMEType,"data":base64.StdEncoding.EncodeToString(d.Data)}}}}},"generationConfig":map[string]any{"responseMimeType":"application/json"}}
	var result struct{Candidates []struct{Content struct{Parts []struct{Text string `json:"text"`} `json:"parts"`} `json:"content"`} `json:"candidates"`}
	endpoint:=fmt.Sprintf("%s/%s:generateContent",strings.TrimRight(g.BaseURL,"/"),g.Model);if err:=postJSON(ctx,g.client(),endpoint,map[string]string{"x-goog-api-key":g.APIKey},body,&result);err!=nil{return domain.RoofPhotoObservation{},err};if len(result.Candidates)==0||len(result.Candidates[0].Content.Parts)==0{return domain.RoofPhotoObservation{},fmt.Errorf("gemini returned no roof observation")};return decodeRoof(result.Candidates[0].Content.Parts[0].Text)
}
func (g Gemini) client() *http.Client {
	if g.HTTP != nil {
		return g.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

type Groq struct {
	APIKey, Model, Endpoint string
	HTTP                    *http.Client
}

func (g Groq) Name() string { return "groq" }
func (g Groq) ExtractBill(ctx context.Context, d Document) (domain.BillExtraction, error) {
	if g.APIKey == "" {
		return domain.BillExtraction{}, ErrUnavailable
	}
	dataURL := "data:" + d.MIMEType + ";base64," + base64.StdEncoding.EncodeToString(d.Data)
	body := map[string]any{"model": g.Model, "temperature": 0, "response_format": map[string]string{"type": "json_object"}, "messages": []any{map[string]any{"role": "user", "content": []any{map[string]string{"type": "text", "text": billPrompt}, map[string]any{"type": "image_url", "image_url": map[string]string{"url": dataURL}}}}}}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if g.Endpoint == "" {
		return domain.BillExtraction{}, fmt.Errorf("groq endpoint is not configured")
	}
	if err := postJSON(ctx, g.client(), g.Endpoint, map[string]string{"Authorization": "Bearer " + g.APIKey}, body, &result); err != nil {
		return domain.BillExtraction{}, err
	}
	if len(result.Choices) == 0 {
		return domain.BillExtraction{}, fmt.Errorf("groq returned no extraction")
	}
	return decodeBill(result.Choices[0].Message.Content)
}
func (g Groq) ExtractRoof(ctx context.Context, d Document) (domain.RoofPhotoObservation, error) {
	if g.APIKey==""||g.Endpoint==""{return domain.RoofPhotoObservation{},ErrUnavailable};dataURL:="data:"+d.MIMEType+";base64,"+base64.StdEncoding.EncodeToString(d.Data);body:=map[string]any{"model":g.Model,"temperature":0,"response_format":map[string]string{"type":"json_object"},"messages":[]any{map[string]any{"role":"user","content":[]any{map[string]string{"type":"text","text":roofPrompt},map[string]any{"type":"image_url","image_url":map[string]string{"url":dataURL}}}}}};var result struct{Choices []struct{Message struct{Content string `json:"content"`} `json:"message"`} `json:"choices"`};if err:=postJSON(ctx,g.client(),g.Endpoint,map[string]string{"Authorization":"Bearer "+g.APIKey},body,&result);err!=nil{return domain.RoofPhotoObservation{},err};if len(result.Choices)==0{return domain.RoofPhotoObservation{},fmt.Errorf("groq returned no roof observation")};return decodeRoof(result.Choices[0].Message.Content)
}
func (g Groq) client() *http.Client {
	if g.HTTP != nil {
		return g.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func postJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, in, out any) error {
	b, e := json.Marshal(in)
	if e != nil {
		return e
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, e := client.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		x, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("provider returned %s: %s", resp.Status, strings.TrimSpace(string(x)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func decodeBill(raw string) (domain.BillExtraction, error) {
	var x struct {
		Utility        string             `json:"utility"`
		BillingPeriod  string             `json:"billing_period"`
		TariffCategory string             `json:"tariff_category"`
		Units          float64            `json:"units_kwh"`
		Amount         float64            `json:"amount"`
		Load           float64            `json:"sanctioned_load_kw"`
		Confidence     map[string]float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(raw), &x); err != nil {
		return domain.BillExtraction{}, fmt.Errorf("invalid extraction JSON: %w", err)
	}
	for k, v := range x.Confidence {
		if v < 0 || v > 1 {
			return domain.BillExtraction{}, fmt.Errorf("confidence for %s outside 0..1", k)
		}
	}
	if x.Units < 0 || x.Amount < 0 || x.Load < 0 {
		return domain.BillExtraction{}, fmt.Errorf("negative bill values are invalid")
	}
	return domain.BillExtraction{Utility: x.Utility, BillingPeriod: x.BillingPeriod, TariffCategory: x.TariffCategory, UnitsKWh: x.Units, Amount: x.Amount, SanctionedLoadKW: x.Load, Confidence: x.Confidence}, nil
}

func decodeRoof(raw string) (domain.RoofPhotoObservation, error) {
	var x domain.RoofPhotoObservation
	if err:=json.Unmarshal([]byte(raw),&x);err!=nil{return x,fmt.Errorf("invalid roof JSON: %w",err)}
	if x.ObstacleEstimatePct<0||x.ObstacleEstimatePct>100{return x,fmt.Errorf("obstacle estimate outside 0..100")}
	for k,v:=range x.Confidence{if v<0||v>1{return x,fmt.Errorf("confidence for %s outside 0..1",k)}}
	return x,nil
}
