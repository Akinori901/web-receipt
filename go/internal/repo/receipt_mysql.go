package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Akinori901/web-receipt/go/internal/entity"
)

// ReceiptRepo は usecase.ReceiptRepo の MySQL 実装。
type ReceiptRepo struct {
	db *sql.DB
}

// NewReceiptRepo は依存を注入して組み立てる。
func NewReceiptRepo(db *sql.DB) *ReceiptRepo {
	return &ReceiptRepo{db: db}
}

const selectReceiptColumns = `
	SELECT id, owner_id, recipient, recipient_address, amount, tax_category,
	       registration_number, proviso, issued_on
	FROM receipts`

// Save は領収書を永続化する（新規発行。ID は発番済み前提）。
func (r *ReceiptRepo) Save(ctx context.Context, rc entity.Receipt) (entity.Receipt, error) {
	const q = `
		INSERT INTO receipts
			(id, owner_id, recipient, recipient_address, amount, tax_category,
			 registration_number, proviso, issued_on)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// 任意項目（住所・登録番号・但し書き）は空なら NULL で入れる。
	if _, err := r.db.ExecContext(ctx, q,
		rc.ID.String(),
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
	row := r.db.QueryRowContext(ctx, selectReceiptColumns+` WHERE id = ?`, id.String())
	return scanReceiptRow(row)
}

// ListByOwner は所有者の領収書を絞り込んで発行日の新しい順に返す。
func (r *ReceiptRepo) ListByOwner(ctx context.Context, owner entity.UserID, filter entity.ReceiptFilter) ([]entity.Receipt, error) {
	q := selectReceiptColumns + ` WHERE owner_id = ?`
	args := []any{owner.String()}
	if filter.Recipient != "" {
		q += ` AND recipient LIKE ?`
		args = append(args, "%"+filter.Recipient+"%")
	}
	if filter.IssuedGTE != "" {
		q += ` AND issued_on >= ?`
		args = append(args, filter.IssuedGTE)
	}
	if filter.IssuedLTE != "" {
		q += ` AND issued_on <= ?`
		args = append(args, filter.IssuedLTE)
	}
	q += ` ORDER BY issued_on DESC, created_at DESC`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("領収書一覧取得に失敗: %w", err)
	}
	defer rows.Close()

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
		rawID, rawOwner, rawRecipient, rawTax string
		rawAddress, rawRegistration, rawProviso sql.NullString
		rawAmount                               int64
		rawIssuedOn                             time.Time
	)
	if err := s.Scan(
		&rawID, &rawOwner, &rawRecipient, &rawAddress, &rawAmount, &rawTax,
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

	return entity.NewReceipt(id, owner, recipient, address, amount, tax, registration, proviso, issuedOn), nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
