package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Akinori901/web-receipt/go/internal/entity"
)

// TenantRepo は usecase.TenantRepo の PostgreSQL 実装。
type TenantRepo struct {
	db *sql.DB
}

func NewTenantRepo(db *sql.DB) *TenantRepo {
	return &TenantRepo{db: db}
}

const selectTenantColumns = `
	SELECT id, corporate_name, address, phone, email, registration_number
	FROM tenants`

// FindByID は組織を取得する。見つからなければ entity.ErrTenantNotFound。
func (r *TenantRepo) FindByID(ctx context.Context, id entity.TenantID) (entity.Tenant, error) {
	row := r.db.QueryRowContext(ctx, selectTenantColumns+` WHERE id = $1`, id.String())
	return scanTenantRow(row)
}

// Save は組織を永続化する（新規・更新）。
func (r *TenantRepo) Save(ctx context.Context, t entity.Tenant) (entity.Tenant, error) {
	const q = `
		INSERT INTO tenants (id, corporate_name, address, phone, email, registration_number)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			corporate_name = EXCLUDED.corporate_name,
			address = EXCLUDED.address,
			phone = EXCLUDED.phone,
			email = EXCLUDED.email,
			registration_number = EXCLUDED.registration_number`

	if _, err := r.db.ExecContext(ctx, q,
		t.ID.String(),
		t.CorporateName.String(),
		nullIfEmpty(t.Address.String()),
		nullIfEmpty(t.Phone.String()),
		nullIfEmpty(t.Email.String()),
		nullIfEmpty(t.RegistrationNumber.String()),
	); err != nil {
		return entity.Tenant{}, fmt.Errorf("組織保存に失敗: %w", err)
	}
	return t, nil
}

func scanTenantRow(row *sql.Row) (entity.Tenant, error) {
	var (
		rawID, rawName                         string
		rawAddress, rawPhone, rawEmail, rawReg sql.NullString
	)
	if err := row.Scan(&rawID, &rawName, &rawAddress, &rawPhone, &rawEmail, &rawReg); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Tenant{}, entity.ErrTenantNotFound
		}
		return entity.Tenant{}, fmt.Errorf("組織取得に失敗: %w", err)
	}

	id, err := entity.NewTenantID(rawID)
	if err != nil {
		return entity.Tenant{}, err
	}
	name, err := entity.NewCorporateName(rawName)
	if err != nil {
		return entity.Tenant{}, err
	}
	address, err := entity.NewAddress(rawAddress.String)
	if err != nil {
		return entity.Tenant{}, err
	}
	phone, err := entity.NewPhone(rawPhone.String)
	if err != nil {
		return entity.Tenant{}, err
	}
	email, err := entity.NewOptionalEmail(rawEmail.String)
	if err != nil {
		return entity.Tenant{}, err
	}
	registration, err := entity.NewRegistrationNumber(rawReg.String)
	if err != nil {
		return entity.Tenant{}, err
	}

	return entity.NewTenant(id, name, address, phone, email, registration), nil
}
