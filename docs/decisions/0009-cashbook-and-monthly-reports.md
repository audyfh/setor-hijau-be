# ADR 0009: Cashbook, Sales to Pengepul, and Monthly Reports

## Status
Accepted

## Context
In community waste bank (Bank Sampah Unit / BSU at RW level) operations, financial tracking spans three interrelated activities:
1. **Waste Collection from Residents (Setoran Nasabah)**:
   Residents deposit sorted recyclable waste. The bank sampah credits the nasabah's balance account (`deposits` / `deposit_lines`). Physical cash is rarely handed to residents at deposit time; instead, residents accumulate a balance (*saldo tabungan sampah*).
2. **Sales of Accumulated Recyclables to Commercial Collectors (Penjualan ke Pengepul / Lapak)**:
   Over weeks or a month, accumulated waste is sold wholesale to commercial buyers (*pengepul*). The pengepul pays cash or bank transfer to the bank sampah treasury at wholesale rates. This constitutes the primary operational revenue (*pemasukan kas*) of the bank sampah.
3. **Cash Withdrawals by Residents (Penarikan Saldo Tunai Nasabah)**:
   Residents withdraw their accumulated savings in cash on demand or at periodic disbursement intervals (e.g., menjelang hari raya atau kas dibuka). This reduces their balance and disburses cash from the bank sampah treasury (*pengeluaran kas*).
4. **General Operational Cash Expenses & Grants**:
   The bank sampah also spends cash on operating supplies (e.g. sacks, scales, stationery, transport) and may receive initial seed capital (*modal awal*) or municipal grants (*hibah*).

Previously in M3, `current_balance_idr` was calculated as `SUM(total_amount_idr)` from deposits. In M5, nasabah withdrawals need to be accounted for, cash transactions must be immutably recorded in an append-only cashbook (`cashbook_entries`), financial reconciliation must be verifiable, and pengelola need monthly aggregation dashboards and exportable reports in CSV and PDF.

## Decisions

### 1. Relational Entities for Waste Bank Financial Lifecycle
We introduce four new tenant-scoped tables:
- `sales_to_pengepul`: Records wholesale waste batch sales (`id`, `bank_sampah_id`, `buyer_name`, `sale_date`, `total_weight_grams`, `total_amount_idr`, `notes`, `recorded_by`, `created_at`).
- `sales_to_pengepul_lines`: Itemized material breakdown of each sale (`id`, `sale_id`, `material_type_id`, `weight_grams`, `price_per_kg_idr`, `subtotal_idr`, `created_at`).
- `nasabah_withdrawals`: Records cash withdrawals by nasabah (`id`, `bank_sampah_id`, `nasabah_id`, `amount_idr`, `notes`, `recorded_by`, `created_at`).
- `cashbook_entries`: The central append-only financial treasury ledger (`id`, `bank_sampah_id`, `entry_type`, `category`, `amount_idr`, `reference_id`, `description`, `recorded_by`, `transaction_date`, `created_at`).

### 2. Atomic Cashbook Journaling
Any event that moves treasury money automatically and atomically creates a `cashbook_entries` record within the same database transaction:
- Creating a sale to pengepul atomically creates a `cash_in` entry with category `'penjualan_pengepul'` and `reference_id = sale.id`.
- Creating a nasabah withdrawal atomically creates a `cash_out` entry with category `'penarikan_nasabah'` and `reference_id = withdrawal.id`.
- Manual entries (`entry_type = 'cash_in' | 'cash_out'`) with categories `'operasional'`, `'modal_awal'`, `'hibah_donasi'`, or `'lainnya'` allow recording other administrative cash movements.

### 3. Append-Only Immutability on Cashbook Entries
Similar to `carbon_ledger`, `cashbook_entries` is strictly append-only:
- Database triggers reject `UPDATE` and `DELETE` queries on `cashbook_entries`.
- Errors are corrected via explicit reversing entries (`reversal`) or adjusting entries, preserving an immutable financial audit trail.

### 4. Derived Nasabah Balance with Withdrawals
Per Domain Invariant #2, nasabah balance is mathematically derived:
$$\text{current\_balance\_idr} = \sum \text{deposits.total\_amount\_idr} - \sum \text{nasabah\_withdrawals.amount\_idr}$$
A withdrawal is rejected if `amount_idr > current_balance_idr` (overdraft protection).

### 5. Multi-Tenant Financial Reconciliation
The cashbook reconciliation endpoint verifies that:
- $\sum \text{sales\_to\_pengepul.total\_amount\_idr} = \sum \text{cashbook\_entries}(\text{category = 'penjualan\_pengepul'})$
- $\sum \text{nasabah\_withdrawals.amount\_idr} = \sum \text{cashbook\_entries}(\text{category = 'penarikan\_nasabah'})$
- $\text{cash\_on\_hand} = \sum \text{cash\_in} - \sum \text{cash\_out}$
- Outstanding liability = $\sum \text{nasabah balance}$ across all active nasabah in the tenant.

### 6. Green Software Report Generation
- **CSV Export**: Generated in-memory with UTF-8 BOM (`\xef\xbb\xbf`) for Indonesian Excel compatibility, structured into executive summary, deposits, sales, and cash movements.
- **PDF Export**: Generated using pure Go library `github.com/jung-kurt/gofpdf` without headless browsers, Chromium binaries, or external C dependencies. Follows Green Software principles: sub-second generation, low memory overhead, and minimal file size (~30-50 KB).

## Consequences
- Clean separation between tenant balance liabilities (owed to nasabah) and physical cash in bank treasury.
- Complete financial transparency for RW administrators and residents.
- Zero floating-point arithmetic throughout the financial domain (`BIGINT` Rupiah and `BIGINT` grams).
