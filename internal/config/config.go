package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port, PublicURL, GoogleMapsKey, GeocodingAPIURL        string
	SupabaseURL, SupabasePublishableKey, SupabaseSecretKey string
	NASAPowerURL                                           string
	GeminiAPIKey, GeminiModel, GeminiAPIBaseURL            string
	GroqAPIKey, GroqModel, GroqAPIURL                      string
	IndiaPolicySourceURL, IndiaPolicyVersion               string
	ReportSigningKey                                       string
	LiveSolar                                              bool
}

func Load() Config {
	return Config{
		Port: value("PORT", "8080"), PublicURL: os.Getenv("PUBLIC_URL"),
		GoogleMapsKey: os.Getenv("GOOGLE_MAPS_API_KEY"), GeocodingAPIURL: os.Getenv("GEOCODING_API_URL"), NASAPowerURL: os.Getenv("NASA_POWER_URL"), SupabaseURL: os.Getenv("SUPABASE_URL"),
		SupabasePublishableKey: os.Getenv("SUPABASE_PUBLISHABLE_KEY"), SupabaseSecretKey: os.Getenv("SUPABASE_SECRET_KEY"),
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"), GeminiModel: value("GEMINI_MODEL", "gemini-2.5-flash"), GeminiAPIBaseURL: os.Getenv("GEMINI_API_BASE_URL"),
		GroqAPIKey: os.Getenv("GROQ_API_KEY"), GroqModel: value("GROQ_MODEL", "meta-llama/llama-4-scout-17b-16e-instruct"), GroqAPIURL: os.Getenv("GROQ_API_URL"),
		IndiaPolicySourceURL: os.Getenv("INDIA_POLICY_SOURCE_URL"), IndiaPolicyVersion: value("INDIA_POLICY_VERSION", "screening-v1"),
		ReportSigningKey: os.Getenv("REPORT_SIGNING_KEY"),
		LiveSolar:        boolValue("LIVE_SOLAR_DATA", false),
	}
}

func (c Config) Addr() string { return ":" + c.Port }
func (c Config) Validate() error {
	if _, err := strconv.Atoi(c.Port); err != nil {
		return fmt.Errorf("PORT must be numeric: %w", err)
	}
	if c.LiveSolar && c.NASAPowerURL == "" {
		return fmt.Errorf("NASA_POWER_URL is required when LIVE_SOLAR_DATA=true")
	}
	if c.GeminiAPIKey != "" && c.GeminiAPIBaseURL == "" {
		return fmt.Errorf("GEMINI_API_BASE_URL is required when GEMINI_API_KEY is set")
	}
	if c.GroqAPIKey != "" && c.GroqAPIURL == "" {
		return fmt.Errorf("GROQ_API_URL is required when GROQ_API_KEY is set")
	}
	if c.SupabaseURL != "" && c.PublicURL == "" {
		return fmt.Errorf("PUBLIC_URL is required when Supabase Auth is configured")
	}
	if c.SupabaseURL != "" && len(c.ReportSigningKey) < 32 {
		return fmt.Errorf("REPORT_SIGNING_KEY must be at least 32 characters when Supabase Auth is configured")
	}
	if c.GoogleMapsKey != "" && c.GeocodingAPIURL == "" {
		return fmt.Errorf("GEOCODING_API_URL is required when GOOGLE_MAPS_API_KEY is set")
	}
	return nil
}
func value(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
func boolValue(k string, fallback bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return fallback
	}
	b, e := strconv.ParseBool(v)
	if e != nil {
		return fallback
	}
	return b
}
