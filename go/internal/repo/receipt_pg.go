package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Akinori901/web-receipt/go/internal/entity"
)

// ReceiptRepo は usecase.ReceiptRepo の PostgreSQL 実装。
type ReceiptRepo struct {
	db *sql.DB
}

// NewReceiptRepo は依存を注入して組み立てる。
func NewReceiptRepo(db *sql.DB) *ReceiptRepo {
	return &ReceiptRepo{db: db}
}

const selectReceiptColumns = `
	SELECT id, tenant_id, owner_id, recipient, recipient_address, amount, tax_category,
	       registration_number, proviso, issued_on
	FROM receipts`

// Save は領収書を永続化する（新規発行。ID は発番済み前提）。
func (r *ReceiptRepo) Save(ctx context.Context, rc entity.Receipt) (entity.Receipt, error) {
	const q = `
		INSERT INTO receipts
			(id, tenant_id, owner_id, recipient, recipient_address, amount, tax_category,
			 registration_number, proviso, issued_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	// 任意項目（住所・登録番号・但し書き）は空なら NULL で入れる。
	if _, err := r.db.ExecContext(ctx, q,
		rc.ID.String(),
		rc.TenantID.String(),
		rc.OwnerID.String(),
		rc.Recipient.String(),
		nullIfEmpty(rc.RecipientAddress.String()),
		rc.Amount.Int64(),
		rc.TaxCategory.String(),
		nullIfEmpty(rc.RegistrationNumber.String()),
		nullIfEmpty(rc.Proviso.String()),
		rc.IssuedOn.Time().Format("2006-01-02"),
	); err != nil {
		return entity.Receipt{}, fmt.Errorf("領収書保存に失敗: %w", err)
	}
	return rc, nil
}

// FindByID は ID で領収書を取得する。見つからなければ entity.ErrReceiptNotFound。
func (r *ReceiptRepo) FindByID(ctx context.Context, id entity.ReceiptID) (entity.Receipt, error) {
	row := r.db.QueryRowContext(ctx, selectReceiptColumns+` WHERE id = $1`, id.String())
	return scanReceiptRow(row)
}

// ListByTenant は組織の領収書を絞り込んで発行日の新しい順に返す。
func (r *ReceiptRepo) ListByTenant(ctx context.Context, tenant entity.TenantID, filter entity.ReceiptFilter) ([]entity.Receipt, error) {
	// PostgreSQL は位置プレースホルダ（$1, $2, ...）。条件を足すたびに番号を進める。
	args := []any{tenant.String()}
	q := selectReceiptColumns + ` WHERE tenant_id = $1`
	if filter.Recipient != "" {
		args = append(args, "%"+filter.Recipient+"%")
		q += fmt.Sprintf(` AND recipient LIKE $%d`, len(args))
	}
	if filter.IssuedGTE != "" {
		args = append(args, filter.IssuedGTE)
		q += fmt.Sprintf(` AND issued_on >= $%d`, len(args))
	}
	if filter.IssuedLTE != "" {
		args = append(args, filter.IssuedLTE)
		q += fmt.Sprintf(` AND issued_on <= $%d`, len(args))
	}
	q += ` ORDER BY issued_on DESC, created_at DESC`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("領収書一覧取得に失敗: %w", err)
	}
	// 読み取り用の結果セットを閉じる失敗は結果に影響しない（読み取りの失敗は rows.Err で見る）
	defer func() { _ = rows.Close() }()

	var receipts []entity.Receipt
	for rows.Next() {
		rc, err := scanReceipt(rows)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, rc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("領収書一覧取得に失敗: %w", err)
	}
	return receipts, nil
}

// rowScanner は *sql.Row と *sql.Rows の共通 Scan を抽象化する。
type rowScanner interface {
	Scan(dest ...any) error
}

func scanReceiptRow(row *sql.Row) (entity.Receipt, error) {
	rc, err := scanReceipt(row)
	if errors.Is(err, sql.ErrNoRows) {
		return entity.Receipt{}, entity.ErrReceiptNotFound
	}
	return rc, err
}

// scanReceipt は DB の行を entity へ変換する。この境界で永続化の都合を断ち切る。
func scanReceipt(s rowScanner) (entity.Receipt, error) {
	var (
		rawID, rawTenant, rawOwner, rawRecipient, rawTax string
		rawAddress, rawRegistration, rawProviso          sql.NullString
		rawAmount                                        int64
		rawIssuedOn                                      time.Time
	)
	if err := s.Scan(
		&rawID, &rawTenant, &rawOwner, &rawRecipient, &rawAddress, &rawAmount, &rawTax,
		&rawRegistration, &rawProviso, &rawIssuedOn,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Receipt{}, entity.ErrReceiptNotFound
		}
		return entity.Receipt{}, fmt.Errorf("領収書取得に失敗: %w", err)
	}

	id, err := entity.NewReceiptID(rawID)
	if err != nil {
		return entity.Receipt{}, err
	}
	tenant, err := entity.NewTenantID(rawTenant)
	if err != nil {
		return entity.Receipt{}, err
	}
	owner, err := entity.NewUserID(rawOwner)
	if err != nil {
		return entity.Receipt{}, err
	}
	recipient, err := entity.NewRecipient(rawRecipient)
	if err != nil {
		return entity.Receipt{}, err
	}
	address, err := entity.NewRecipientAddress(rawAddress.String)
	if err != nil {
		return entity.Receipt{}, err
	}
	amount, err := entity.NewMoney(rawAmount)
	if err != nil {
		return entity.Receipt{}, err
	}
	tax, err := entity.NewTaxCategory(rawTax)
	if err != nil {
		return entity.Receipt{}, err
	}
	registration, err := entity.NewRegistrationNumber(rawRegistration.String)
	if err != nil {
		return entity.Receipt{}, err
	}
	proviso, err := entity.NewProviso(rawProviso.String)
	if err != nil {
		return entity.Receipt{}, err
	}
	issuedOn, err := entity.NewIssuedOn(rawIssuedOn)
	if err != nil {
		return entity.Receipt{}, err
	}

	return entity.NewReceipt(id, tenant, owner, recipient, address, amount, tax, registration, proviso, issuedOn), nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
