# Domain

## Glossary
| Term | Meaning |
|---|---|
| Bank sampah | Community waste bank where residents deposit sorted waste and receive a balance. Also the tenant in this system. |
| Bank Sampah Unit (BSU) | Community-level waste bank (RT, RW, kelurahan, or village) as defined by Permen LHK 14/2021. Setor Hijau targets RW-level BSU. |
| RW / RT | Rukun Warga / Rukun Tetangga: neighborhood administrative units. |
| Nasabah | Customer: a resident who deposits waste. |
| Pengelola | Manager: officer and administrator of the bank sampah. |
| Setoran | A deposit of sorted waste, one or more material lines. |
| Jenis sampah | Material type (e.g. plastic bottles, cardboard, aluminum cans). |
| Harga | Price per unit weight paid to nasabah for a material, set by the Pengelola. |
| Pengepul | Collector/buyer to whom the bank sampah sells accumulated material. |
| Saldo | Rupiah balance of a nasabah. |
| Buku kas | Cashbook: automatic record of money in and out of the bank sampah. |
| Struk digital | Digital receipt for a deposit. |
| CO2e | Carbon dioxide equivalent: greenhouse gas emissions expressed as CO2. |
| Carbon Ledger | Append-only record of estimated CO2e avoided per deposit, with factor version and source. |
| Tabungan karbon | A nasabah's accumulated estimated avoided CO2e. |
| Emission factor | CO2e avoided per unit weight of a material when recycled instead of landfilled (from WARM). |
| WARM | US EPA Waste Reduction Model, version 16 used for factors. |
| SWDM | Sustainable Web Design Model, version 4, used for the app's digital footprint. |
| Net Impact Meter | Total estimated avoided CO2e minus the app's estimated digital footprint. |
| TPA | Tempat Pemrosesan Akhir: final landfill. |

## Entities (conceptual; schema lives in `DATA-MODEL.md`)
bank_sampah (tenant) -> users (Pengelola, Nasabah) -> deposits -> deposit_lines -> carbon_ledger entries. Also: materials, material_prices (history), cashbook entries, emission_factors (versioned), digital_footprint measurements, refresh tokens.

## Business invariants
1. A nasabah and all their data belong to exactly one bank sampah.
2. A nasabah's balance always equals the sum of their balance entries (deposits credit, withdrawals debit); it is derived from entries, not edited directly.
3. Deposits and carbon ledger rows are immutable. A mistake is fixed by a reversing entry plus a new correct entry.
4. A deposit line's rupiah value uses the material price in effect at the time of the deposit and is stored on the line. Later price changes never alter past deposits.
5. A carbon ledger row references the emission factor version and source used when it was created. Updating factors never rewrites old rows.
6. Weight is stored in integer grams, money in integer rupiah. No floats.
7. Replaying the same deposit request (same idempotency key) creates no second deposit and returns the original result.
8. Client-recorded time and server-received time are stored separately.
9. Net impact = sum(avoided CO2e) - estimated digital footprint over the same scope and period; both inputs remain inspectable.
10. The Pengelola can only see and change data of their own bank sampah; a Nasabah can only see their own data.
