# SolarSense pending work

The first implementation is committed and the local assessment flow is working. The items below require external credentials, policy review, or production hardening.

## Required before a real deployment

- Create a Supabase project and apply `supabase/migrations/202609150001_initial_schema.sql`.
- Configure Google as a Supabase Auth provider and add the production callback URL to the redirect allowlist.
- Set `PUBLIC_URL`, `SUPABASE_URL`, `SUPABASE_PUBLISHABLE_KEY`, and a random `REPORT_SIGNING_KEY` with at least 32 characters.
- Set the configured NASA POWER endpoint and decide whether live solar data should be enabled.
- Add Gemini and Groq credentials plus their configured endpoint and model values.
- Add the geocoding provider URL/key if city and pincode search should resolve coordinates automatically; GPS and manual entry work without it.
- Deploy the Docker image to a container host and configure HTTPS. HTTPS is required for browser geolocation and secure cookies.

## Product and data work still needed

- Review and activate India policy packs with official tariff, export-metering, sanctioned-load, subsidy, and state-incentive sources.
- Add reviewed DISCOM adapters and fixtures for each Indian region marked verified; uncovered locations remain preliminary.
- Populate reviewed searchable knowledge documents and embeddings in Supabase.
- Add country adapters for currencies, tariffs, export rules, and incentives before claiming verified coverage outside India.
- Add hourly or measured household-load inputs for higher-confidence battery simulations.
- Add installer/site-survey handoff for roof dimensions, structural checks, shadows, and utility approval.

## Known implementation follow-ups

- PDF bills are sent directly to Gemini; add bounded PDF page rendering before relying on Groq as a PDF fallback.
- Uploaded bill and roof files are processed in memory; persist them temporarily in the private Supabase bucket only if audit/reprocessing is required, with automatic deletion within 24 hours.
- Add refresh-token/session persistence and sign-out UI for long-lived Supabase sessions.
- Expand Telugu and Hindi coverage from the assessment headings to every field label, validation message, result explanation, and saved-report page.
- Add integration tests against a disposable Supabase project and mocked NASA, geocoding, Gemini, and Groq endpoints.
- Run the container build in CI or a Docker-enabled environment; Docker is not installed in the current workspace.
