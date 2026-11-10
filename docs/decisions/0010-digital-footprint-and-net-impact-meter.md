# ADR 0010: Digital Footprint Accounting (SWDM v4) and Net Impact Meter

## Status
Accepted

## Context
Setor Hijau is designed under Green Software Engineering principles and academic sustainability honesty. While waste banks avoid substantial greenhouse gas emissions by diverting recyclables from landfills to recycling facilities, running a web platform itself consumes electricity across datacenters, transmission networks, and user client devices.

To present an honest, comprehensive environmental balance sheet rather than mere greenwashing:
1. We must calculate the app's own digital carbon footprint.
2. We must compute the **Net Impact Meter**:
   $$\text{Net Impact } (g CO_2e) = \sum \text{Avoided } CO_2e \text{ (WARM v16)} - \sum \text{Digital Footprint } CO_2e \text{ (SWDM v4)}$$
3. We must resolve the open question from the project roadmap: *Where CO2.js runs and how the API receives measurements*.
4. We must provide public transparency via a dedicated "Metode & Sumber" endpoint with zero authentication barriers so that anyone can inspect formulas, factors, citations, and reproducible figures.

## Decisions

### 1. Model Selection: Sustainable Web Design Model (SWDM) Version 4
We adopt the **Sustainable Web Design Model (SWDM) v4** (developed by Wholegrain Digital, Mightybytes, Medina Works, and standardized in CO2.js >= 0.18):
- **Operational Energy**: $0.057 \text{ kWh / GB}$
- **Embodied Energy (Hardware/Lifecycle)**: $0.015 \text{ kWh / GB}$
- **Total Energy**: $0.072 \text{ kWh / GB}$
- **Global Average Grid Carbon Intensity**: $494 \text{ g } CO_2e / \text{kWh}$
- **Conversion Factor**:
  $$0.072 \times 494 = 35.568 \text{ g } CO_2e / \text{GB} = 35.568 \times 10^{-9} \text{ g } CO_2e / \text{byte}$$
- **System Segments Attribution**:
  - Data center: $15\%$
  - Network: $14\%$
  - User device: $52\%$
  - Embodied hardware: $19\%$

### 2. Resolution of Open Question: Client Telemetry with Deterministic Server Calculation
- The frontend tracks data bytes transferred during user sessions (via standard W3C `PerformanceResourceTiming` APIs or CO2.js wrapper).
- The client dispatches telemetry payloads to `POST /api/v1/public/impact/telemetry` containing: `page_path`, `data_bytes_transferred`, optional `bank_sampah_id`, `visitor_type`, and `session_id`.
- The backend verifies and computes the exact digital emissions on the server using deterministic integer fixed-point math:
  $$\text{co2e\_micrograms} = \left\lfloor \frac{B \times 35.568 + 500.000}{1.000.000} \right\rfloor$$
  $$\text{co2e\_grams} = \left\lfloor \frac{\text{co2e\_micrograms} + 500.000}{1.000.000} \right\rfloor$$
- This guarantees that stored inputs (`data_bytes_transferred`) remain inspectable and reproducible even if client JavaScript calculations vary.

### 3. Public Endpoints for Academic Transparency
- `GET /api/v1/public/methods`: Delivers the complete methodology, parameter constants, WARM v16 factors, SWDM v4 equations, assumptions, and citations without authentication.
- `GET /api/v1/public/impact`: Returns platform-wide net impact, avoided emissions, digital footprint, sustainability ratio, and category breakdowns.
- `GET /api/v1/public/impact/tenants/:code`: Returns public net impact overview for a specific bank sampah by its URL-friendly code.
- `GET /api/v1/impact/tenant`: Returns authenticated net impact overview for the current tenant.

### 4. Integer Scaling (Hard Rule #3)
All digital footprint metrics are stored and computed using 64-bit integers (`int64` / `BIGINT`):
- `data_bytes_transferred` in integer bytes (`BIGINT`).
- `co2e_micrograms` in integer micrograms ($\mu\text{g } CO_2e$) to preserve precision for small HTTP requests.
- `co2e_grams` in integer grams ($g CO_2e$) rounded half-up for higher-level metrics.

## Consequences
- The Net Impact calculation is 100% reproducible and auditable from stored inputs.
- Adheres to Green Software Engineering by maintaining a lightweight backend implementation with minimal overhead.
- All carbon fields continue to carry the `estimated_` prefix per Hard Rule #9.
