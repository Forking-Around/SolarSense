# SolarSense

SolarSense is a server-rendered Go application for transparent residential rooftop-solar and battery screening. Its calculations are deterministic; AI is limited to reading user-provided documents and explaining confirmed results.

## Run locally

```sh
cp .env.example .env
# For an offline screening profile, set LIVE_SOLAR_DATA=false.
set -a; source .env; set +a
go run ./cmd/server
```

Open the configured `PUBLIC_URL`. The manual assessment works without Supabase or AI credentials when `LIVE_SOLAR_DATA=false`.

## Configuration

All external URLs are supplied through environment variables. See `.env.example` for NASA POWER, Supabase, Gemini and Groq endpoint variables. Secrets must remain server-side.

To enable Google sign-in, configure Google as a Supabase social provider, add the application callback URL to the Supabase redirect allowlist, and supply the Supabase project URL and publishable key.

Apply `supabase/migrations` to a Supabase project before enabling saved assessments or private uploads. The schema enables RLS on every public table and limits user-owned records and storage paths to their owner.

## Checks

```sh
gofmt -w cmd internal
go test ./...
go vet ./...
```

This is a screening estimate. Roof measurements, shading, policy eligibility, utility approval, and installation design require professional verification.
