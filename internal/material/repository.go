package material

import (
	"context"
	"errors"
	"time"

	"github.com/audyfh/setor-hijau-be/internal/platform/db"
	"github.com/audyfh/setor-hijau-be/internal/platform/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrMaterialNotFound indicates the material type was not found.
	ErrMaterialNotFound = errors.New("jenis sampah tidak ditemukan")
)

// Repository defines database operations for materials and historical prices.
type Repository interface {
	ExecTx(ctx context.Context, fn func(q *sqlc.Queries) error) error
	CreateMaterialType(ctx context.Context, q *sqlc.Queries, params sqlc.CreateMaterialTypeParams) (*sqlc.MaterialType, error)
	UpdateMaterialType(ctx context.Context, q *sqlc.Queries, params sqlc.UpdateMaterialTypeParams) (*sqlc.MaterialType, error)
	GetMaterialTypeByID(ctx context.Context, id, bankSampahID uuid.UUID) (*Material, error)
	ListMaterialTypes(ctx context.Context, bankSampahID uuid.UUID, activeOnly bool) ([]*Material, error)
	CreateMaterialPrice(ctx context.Context, q *sqlc.Queries, params sqlc.CreateMaterialPriceParams) (*sqlc.MaterialPrice, error)
	CloseActiveMaterialPrice(ctx context.Context, q *sqlc.Queries, effectiveTo time.Time, materialTypeID, bankSampahID uuid.UUID) error
	ListMaterialPricesHistory(ctx context.Context, materialTypeID, bankSampahID uuid.UUID) ([]*PriceHistoryEntry, error)
}

type pgRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

// NewRepository creates a new PostgreSQL repository for materials.
func NewRepository(pool *pgxpool.Pool, queries *sqlc.Queries) Repository {
	return &pgRepository{
		pool:    pool,
		queries: queries,
	}
}

func (r *pgRepository) ExecTx(ctx context.Context, fn func(q *sqlc.Queries) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	q := r.queries.WithTx(tx)
	if err := fn(q); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *pgRepository) CreateMaterialType(ctx context.Context, q *sqlc.Queries, params sqlc.CreateMaterialTypeParams) (*sqlc.MaterialType, error) {
	if q == nil {
		q = r.queries
	}
	m, err := q.CreateMaterialType(ctx, params)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *pgRepository) UpdateMaterialType(ctx context.Context, q *sqlc.Queries, params sqlc.UpdateMaterialTypeParams) (*sqlc.MaterialType, error) {
	if q == nil {
		q = r.queries
	}
	m, err := q.UpdateMaterialType(ctx, params)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *pgRepository) GetMaterialTypeByID(ctx context.Context, id, bankSampahID uuid.UUID) (*Material, error) {
	row, err := r.queries.GetMaterialTypeByID(ctx, sqlc.GetMaterialTypeByIDParams{
		ID:           db.ToPGUUID(id),
		BankSampahID: db.ToPGUUID(bankSampahID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMaterialNotFound
		}
		return nil, err
	}

	var priceEffFrom *time.Time
	if row.PriceEffectiveFrom.Valid {
		t := db.FromPGTimestamptz(row.PriceEffectiveFrom)
		priceEffFrom = &t
	}

	priceID := db.FromPGUUID(row.PriceID)
	priceVal := row.PricePerKgIdr

	return &Material{
		ID:                 db.FromPGUUID(row.ID),
		BankSampahID:       db.FromPGUUID(row.BankSampahID),
		Name:               row.Name,
		Category:           row.Category,
		Unit:               row.Unit,
		Description:        db.FromPGText(row.Description),
		IsActive:           row.IsActive,
		CreatedAt:          db.FromPGTimestamptz(row.CreatedAt),
		UpdatedAt:          db.FromPGTimestamptz(row.UpdatedAt),
		PriceID:            priceID,
		PricePerKgIDR:      priceVal,
		PriceEffectiveFrom: priceEffFrom,
	}, nil
}

func (r *pgRepository) ListMaterialTypes(ctx context.Context, bankSampahID uuid.UUID, activeOnly bool) ([]*Material, error) {
	rows, err := r.queries.ListMaterialTypes(ctx, sqlc.ListMaterialTypesParams{
		BankSampahID: db.ToPGUUID(bankSampahID),
		Column2:      activeOnly,
	})
	if err != nil {
		return nil, err
	}

	results := make([]*Material, len(rows))
	for i, row := range rows {
		var priceEffFrom *time.Time
		if row.PriceEffectiveFrom.Valid {
			t := db.FromPGTimestamptz(row.PriceEffectiveFrom)
			priceEffFrom = &t
		}

		priceID := db.FromPGUUID(row.PriceID)
		priceVal := row.PricePerKgIdr

		results[i] = &Material{
			ID:                 db.FromPGUUID(row.ID),
			BankSampahID:       db.FromPGUUID(row.BankSampahID),
			Name:               row.Name,
			Category:           row.Category,
			Unit:               row.Unit,
			Description:        db.FromPGText(row.Description),
			IsActive:           row.IsActive,
			CreatedAt:          db.FromPGTimestamptz(row.CreatedAt),
			UpdatedAt:          db.FromPGTimestamptz(row.UpdatedAt),
			PriceID:            priceID,
			PricePerKgIDR:      priceVal,
			PriceEffectiveFrom: priceEffFrom,
		}
	}
	return results, nil
}

func (r *pgRepository) CreateMaterialPrice(ctx context.Context, q *sqlc.Queries, params sqlc.CreateMaterialPriceParams) (*sqlc.MaterialPrice, error) {
	if q == nil {
		q = r.queries
	}
	p, err := q.CreateMaterialPrice(ctx, params)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *pgRepository) CloseActiveMaterialPrice(ctx context.Context, q *sqlc.Queries, effectiveTo time.Time, materialTypeID, bankSampahID uuid.UUID) error {
	if q == nil {
		q = r.queries
	}
	return q.CloseActiveMaterialPrice(ctx, sqlc.CloseActiveMaterialPriceParams{
		EffectiveTo:    db.ToPGTimestamptz(effectiveTo),
		MaterialTypeID: db.ToPGUUID(materialTypeID),
		BankSampahID:   db.ToPGUUID(bankSampahID),
	})
}

func (r *pgRepository) ListMaterialPricesHistory(ctx context.Context, materialTypeID, bankSampahID uuid.UUID) ([]*PriceHistoryEntry, error) {
	rows, err := r.queries.ListMaterialPricesHistory(ctx, sqlc.ListMaterialPricesHistoryParams{
		MaterialTypeID: db.ToPGUUID(materialTypeID),
		BankSampahID:   db.ToPGUUID(bankSampahID),
	})
	if err != nil {
		return nil, err
	}

	results := make([]*PriceHistoryEntry, len(rows))
	for i, row := range rows {
		var effTo *time.Time
		if row.EffectiveTo.Valid {
			t := db.FromPGTimestamptz(row.EffectiveTo)
			effTo = &t
		}

		var createdBy *uuid.UUID
		if row.CreatedBy.Valid {
			u := db.FromPGUUID(row.CreatedBy)
			createdBy = &u
		}

		results[i] = &PriceHistoryEntry{
			ID:             db.FromPGUUID(row.ID),
			MaterialTypeID: db.FromPGUUID(row.MaterialTypeID),
			BankSampahID:   db.FromPGUUID(row.BankSampahID),
			PricePerKgIDR:  row.PricePerKgIdr,
			EffectiveFrom:  db.FromPGTimestamptz(row.EffectiveFrom),
			EffectiveTo:    effTo,
			CreatedBy:      createdBy,
			CreatedAt:      db.FromPGTimestamptz(row.CreatedAt),
		}
	}
	return results, nil
}
