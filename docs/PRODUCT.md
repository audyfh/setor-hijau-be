# Product

## Problem
RW-level bank sampah typically record deposits and balances in handwritten books. This is error-prone, hard to audit, and slow to report. Residents see only the rupiah value of their waste, with no view of its environmental contribution.

## Solution
A multi-tenant web platform. A bank sampah registers in the browser (no install, no own server). Deposits are recorded per nasabah, balances and cashbook are automatic, and each deposit also records estimated avoided CO2e with a traceable method.

## Roles
| Role | Who | Can do |
|---|---|---|
| Pengelola | Officer and administrator of one bank sampah | Register the bank sampah, manage nasabah, set material types and prices, record deposits (including offline), view cashbook and monthly dashboard, export CSV/PDF |
| Nasabah | RW resident with an account at one bank sampah | View rupiah balance and carbon savings, deposit history with digital receipt (struk), sorting guide |
| Public (no login) | Anyone | "Metode & Sumber" page and Net Impact Meter |

Only Nasabah and Pengelola exist. No separate super-admin feature is planned; platform operations happen outside the app.

## Authentication
- Regular login (identifier + password) for both roles. Primary.
- QR code + PIN login for Nasabah: optional extra, not required.
- The Nasabah QR code is also used by the Pengelola to select a nasabah quickly when recording a deposit.

## Features (in scope)
1. Bank sampah registration (creates a tenant) with data and access isolated per bank sampah and role.
2. Nasabah management by the Pengelola.
3. Material types and prices (price history; old deposits keep the price used at the time).
4. Deposit recording: nasabah, material, weight in grams, rupiah value. Offline-tolerant and idempotent.
5. Rupiah balance per nasabah, digital receipt, deposit history.
6. Automatic cashbook (buku kas), including sales to pengepul (collectors).
7. Monthly dashboard and CSV/PDF report export.
8. Carbon Ledger: estimated CO2e avoided per deposit, with factor version and source, auditable.
9. Net Impact Meter: avoided CO2e minus the app's estimated digital footprint (net impact).
10. Public "Metode & Sumber" page: formulas, emission factors, sources.
11. Sorting guide content for residents.

## Out of scope
Digital payments, marketplace integration, kelurahan-level dashboard, pickup scheduling, leaderboard. Also: organic waste carbon accounting, direct electricity measurement of servers or devices, additional roles.

## Constraints and honesty rules
- Carbon values are **estimates**: WARM factors are based on US conditions. Name API fields and UI labels accordingly (e.g. `estimated_co2e_avoided`).
- Digital footprint is estimated from data transfer per visit using SWDM v4, not measured.
- CO2e covers only inorganic recyclable waste diverted from landfill.

## Success criteria (project level)
- A Pengelola can run a full month of operations (deposits, balances, cashbook, report) in the system.
- Every carbon figure can be traced to material, weight, factor, factor version, and source.
- Data of one bank sampah is never visible to another.
- The app stays lightweight (Green Software Engineering): small payloads, caching, no wasteful assets.
