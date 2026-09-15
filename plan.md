# SolarSense Implementation Plan

## Summary

Build SolarSense as a Go web application that answers whether rooftop solar and battery backup make sense for a household.

The first implementation will support:

- Server-rendered Go pages with HTMX.
- Supabase PostgreSQL, Google authentication, and private file storage.
- Gemini for bill/roof-photo interpretation with Groq fallback.
- GPS, location search, and Indian pincode entry.
- Area-specific expected solar generation.
- Roof-space, orientation, tilt, obstacles, and time-of-day shading analysis.
- Separate financial recommendations for on-grid solar and backup recommendations for batteries.
- Verified Indian policies first, with a worldwide-ready country adapter architecture.
- English, Telugu, and Hindi.

## Implementation Changes

### 1. Repository rules and foundation

Create `AGENTS.md` before other implementation files. It will establish:

- Go, `net/http`, `html/template`, HTMX, PostgreSQL, and Supabase as the approved stack.
- Package boundaries between web handlers, solar physics, tariffs, batteries, AI providers, and persistence.
- Deterministic calculations as the only source of financial and engineering outputs.
- Gemini as primary and Groq as fallback; AI output must pass typed validation.
- Official-source citations, effective dates, and review status for every active policy rule.
- No invented tariffs, subsidies, outage statistics, roof dimensions, or bill values.
- Supabase migration, RLS, private-storage, testing, formatting, and secret-management requirements.
- Instructions to search approved coding skills before adding unfamiliar integrations and document any adopted skill.

Then scaffold:

- Go application with HTML templates, HTMX fragments, localized message catalogs, embedded static assets, configuration validation, structured logging, and graceful shutdown.
- Docker-based runtime and local development configuration.
- Supabase migrations for users, assessments, locations, bill extractions, roof observations, reports, policy packs, knowledge documents, solar-data cache, and provider audit metadata.
- Ownership-based RLS for user records and private storage. Service credentials remain server-only.

### 2. Assessment and roof intelligence

The assessment begins immediately with simple questions.

**Location**

- “Use my location” requests browser GPS permission after a user action.
- City/village search and Indian pincode entry remain available.
- Reverse-geocode coordinates and require the user to confirm the installation location.
- Keep current-device location separate from installation location so users can assess another property.
- Ask the user to confirm the electricity provider because geographic lookup may return multiple DISCOMs.

**Roof space**

Collect:

- RCC flat, metal sheet, tiled/sloped, apartment/shared terrace, or other.
- Rough dimensions or familiar choices such as full 800 sq.ft roof, partial roof, balcony, or shared terrace.
- Areas occupied by water tanks, stair rooms, vents, parapets, antennas, and access paths.
- Roof ownership and permission status.
- Optional roof photos after Google sign-in.

Calculate both gross and usable area. Reserve configurable clearance and maintenance-access space, then compare usable area with module-layout requirements. Never claim precise measurements from a photograph without a known-size reference; show an estimate range and require installer verification.

**Direction, angle, and sunlight**

Use plain visual questions:

- Flat or sloped roof.
- Direction faced, if known: north, south, east, west, mixed, or “I don’t know.”
- Morning sun, midday sun, and evening sun as selectable visual time blocks.
- Seasonal shading and nearby trees/buildings.
- Whether shadows cross only part of the roof or most of it.

Gemini may extract visible obstacles, roof type, and likely shading from optional photos; Groq handles supported image fallback. The user confirms all extracted observations.

The engine will:

- Use hourly solar-climatology profiles for the selected coordinates.
- Apply shading losses only to the affected time blocks.
- Apply orientation and tilt factors through tested lookup/model functions.
- Evaluate multiple panel zones separately when roof faces differ.
- Produce conservative, expected, and optimistic generation estimates.
- Warn when a professional shade study or site survey could materially change the answer.

### 3. Solar, battery, and financial engine

**Expected area generation**

Before recommending a system, show:

- Expected annual generation per installed kW.
- Month-by-month production for the selected location.
- Seasonal high and low months.
- The solar-data source, observation period, retrieval date, and confidence.

After sizing, show:

- Recommended capacity and usable roof area required.
- Monthly and annual system generation.
- Self-consumed, exported, and grid-imported energy.
- Expected remaining electricity bill.
- Generation range reflecting shade, direction, weather, and modelling uncertainty.

Use NASA POWER as the initial solar source and cache normalized datasets. Source adapters allow later use of NREL, PVGIS, or country-specific datasets.

**Solar calculations**

Evaluate candidate capacities against:

- Annual and monthly consumption.
- Day/night usage profile.
- Usable roof zones.
- Local tariff and export rules.
- Sanctioned-load or connection restrictions.
- Incentive limits.
- Expected curtailment and export value.

Compare the existing grid bill with each candidate. Include system losses, module degradation, inverter replacement, maintenance, fixed charges, tariff escalation scenarios, and incentive reimbursement timing.

**Battery intelligence**

Ask only when backup or battery analysis can affect the result:

- Outage frequency and typical duration.
- Daytime versus nighttime outages.
- Essential appliances.
- Existing inverter, battery, or generator.
- Desired backup duration.
- Whether the priority is savings, backup, or both.

Return two independent battery conclusions:

- **Battery for savings:** based on tariff arbitrage, avoided imports, export value, degradation, and replacement cost.
- **Battery for backup:** based on outage frequency, essential load, usable capacity, discharge power, and desired autonomy.

Location-level reliability evidence may adjust confidence, but it never replaces the household’s reported experience. Hyderabad and Kovvali are validation scenarios rather than fixed assumptions:

- Rare reported outages should usually make an on-grid system preferable financially.
- Frequent long outages can justify battery backup even when battery payback is unattractive.
- Reversing the outage inputs must be able to reverse the backup recommendation.

The report will explain statements such as: “Solar makes financial sense; a battery does not recover its cost under your current tariff, but a 5 kWh battery would provide about four hours for your selected essential loads.”

### 4. AI, searchable knowledge, and public interfaces

**Gemini and Groq**

Create shared Go interfaces for:

- Bill extraction.
- Roof-photo observations.
- Translation/query normalization.
- Evidence-grounded report explanations.

Provider order:

1. Gemini primary.
2. Retry once for a transient failure.
3. Groq using a model verified to support the supplied image or structured response.
4. Manual entry or deterministic explanation if both fail.

Images and PDFs are normalized and size/page limited. Extracted bill fields include provider, billing period, units, tariff category, sanctioned load, charges, and confidence. Users must confirm uncertain fields before calculation.

**Searchable product knowledge**

Store source-backed documents for tariffs, incentives, metering, connection rules, batteries, and roof guidance.

- Use PostgreSQL full-text search for exact names, utilities, pincodes, and policy terminology.
- Add `pgvector` semantic retrieval for natural-language questions.
- Filter retrieval by country, state, utility, effective date, language, and review status.
- Keep searchable documents separate from executable policy rules.
- AI explanations may cite retrieved evidence but cannot alter calculations.
- Add an administrative import/review command that activates a policy version only after validation fixtures pass.

**Web interfaces**

Provide server-rendered routes and HTMX endpoints for:

- Assessment questions and adaptive follow-ups.
- Location search and coordinate confirmation.
- Authenticated bill/roof-photo upload and extraction confirmation.
- Scenario calculation and comparison.
- Knowledge search.
- Saved report list, view, print, and delete.
- Google OAuth callback and secure session lifecycle through Supabase Auth.

Core Go types include:

- `AssessmentInput`
- `LocationContext`
- `RoofProfile` and `RoofZone`
- `SunlightWindow`
- `OutageProfile`
- `BillExtraction`
- `PolicyPack`
- `KnowledgeDocument`
- `SolarScenario`
- `BatteryScenario`
- `AssessmentReport`

Every saved report records input provenance, calculation assumptions, solar dataset version, policy version, and engine version.

## Test Plan

- GPS granted, denied, timed out, and unavailable; manual city/village/pincode fallback.
- Duplicate place names, wrong-country pincodes, and assessment of a location different from the user’s current position.
- Flat, east/west split, tilted, unknown-direction, partially shaded, and heavily obstructed roofs.
- Morning-only, midday-only, evening-only, partial-zone, and seasonal shading.
- Photo extraction with and without a scale reference; unreadable images and conflicting user answers.
- Usable-area constraints, maintenance clearance, apartment/shared-roof permission, and insufficient roof space.
- Monthly generation totals, orientation factors, time-block shade losses, degradation, and uncertainty bounds.
- Tariff slabs, fixed charges, exports, subsidies, connection rules, and missing/unverified policy data.
- Rare and frequent outage scenarios in both Hyderabad and Kovvali.
- Battery energy balance, charge/discharge power, reserve, degradation, replacement, essential-load runtime, and incremental payback.
- Gemini success, Gemini-to-Groq fallback, schema-invalid responses, both providers unavailable, and manual correction.
- Google sign-in return flow, CSRF, cross-user report rejection, private uploads, and scheduled upload deletion.
- English, Telugu, and Hindi assessment/report rendering on mobile and desktop.

## Assumptions and Defaults

- V1 covers residential cash purchases; loans and installer quotations are excluded.
- India receives verified policy coverage first. Other countries use preliminary user-supplied tariff inputs until reviewed adapters are added.
- Manual assessments remain anonymous. Bill uploads, roof photos, and saved reports require Google sign-in.
- Roof photos improve obstacle identification but do not replace physical measurement or an installer survey.
- User-reported outages take priority over general city/village assumptions.
- A firm verdict requires verified tariff/export rules and adequate roof/input confidence; otherwise the result is labelled preliminary.
- Production deployment is packaged as a containerized Go service, with Supabase providing PostgreSQL, Auth, and Storage.
