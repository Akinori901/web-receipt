package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Akinori901/web-receipt/go/internal/entity"
)

// HandoverRepo は usecase.HandoverRepo の PostgreSQL 実装。
type HandoverRepo struct {
	db *sql.DB
}

func NewHandoverRepo(db *sql.DB) *HandoverRepo {
	return &HandoverRepo{db: db}
}

const selectHandoverColumns = `
	SELECT receipt_id, view_id, view_password, url_view_count, pdf_count,
	       recipient_confirmed, recipient
	FROM handovers`

// Save は受け渡しを永続化する（新規・宛名確定の更新）。
func (r *HandoverRepo) Save(ctx context.Context, h entity.Handover) (entity.Handover, error) {
	const q = `
		INSERT INTO handovers
			(receipt_id, view_id, view_password, url_view_count, pdf_count,
			 recipient_confirmed, recipient)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (receipt_id) DO UPDATE SET
			recipient_confirmed = EXCLUDED.recipient_confirmed,
			recipient = EXCLUDED.recipient`

	if _, err := r.db.ExecContext(ctx, q,
		h.ReceiptID.String(),
		h.ViewID.String(),
		h.ViewPassword.String(),
		h.URLViewCount,
		h.PDFCount,
		h.RecipientConfirmed,
		nullIfEmpty(h.Recipient.String()),
	); err != nil {
		return entity.Handover{}, fmt.Errorf("受け渡し保存に失敗: %w", err)
	}
	return h, nil
}

// FindByReceiptID は領収書IDで受け渡しを取得する。
func (r *HandoverRepo) FindByReceiptID(ctx context.Context, receiptID entity.ReceiptID) (entity.Handover, error) {
	row := r.db.QueryRowContext(ctx, selectHandoverColumns+` WHERE receipt_id = $1`, receiptID.String())
	return scanHandoverRow(row)
}

// FindByViewID は閲覧IDで受け渡しを取得する。
func (r *HandoverRepo) FindByViewID(ctx context.Context, viewID entity.ViewID) (entity.Handover, error) {
	row := r.db.QueryRowContext(ctx, selectHandoverColumns+` WHERE view_id = $1`, viewID.String())
	return scanHandoverRow(row)
}

// IncrementURLView は URL 閲覧数を原子的に +1 する。
func (r *HandoverRepo) IncrementURLView(ctx context.Context, viewID entity.ViewID) error {
	return r.increment(ctx, "url_view_count", viewID)
}

// IncrementPDF は PDF 出力数を原子的に +1 する。
func (r *HandoverRepo) IncrementPDF(ctx context.Context, viewID entity.ViewID) error {
	return r.increment(ctx, "pdf_count", viewID)
}

func (r *HandoverRepo) increment(ctx context.Context, column string, viewID entity.ViewID) error {
	// column は内部固定値（外部入力でない）なので SQL インジェクションの余地はない。
	q := fmt.Sprintf(`UPDATE handovers SET %s = %s + 1 WHERE view_id = $1`, column, column)
	res, err := r.db.ExecContext(ctx, q, viewID.String())
	if err != nil {
		return fmt.Errorf("カウント加算に失敗: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return entity.ErrHandoverNotFound
	}
	return nil
}

func scanHandoverRow(row *sql.Row) (entity.Handover, error) {
	var (
		rawReceiptID, rawViewID, rawPassword string
		urlCount, pdfCount                   int
		confirmed                            bool
		rawRecipient                         sql.NullString
	)
	if err := row.Scan(&rawReceiptID, &rawViewID, &rawPassword, &urlCount, &pdfCount, &confirmed, &rawRecipient); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Handover{}, entity.ErrHandoverNotFound
		}
		return entity.Handover{}, fmt.Errorf("受け渡し取得に失敗: %w", err)
	}

	receiptID, err := entity.NewReceiptID(rawReceiptID)
	if err != nil {
		return entity.Handover{}, err
	}
	viewID, err := entity.NewViewID(rawViewID)
	if err != nil {
		return entity.Handover{}, err
	}
	password, err := entity.NewViewPassword(rawPassword)
	if err != nil {
		return entity.Handover{}, err
	}

	h := entity.NewHandover(receiptID, viewID, password)
	h.URLViewCount = urlCount
	h.PDFCount = pdfCount
	h.RecipientConfirmed = confirmed
	if rawRecipient.Valid && rawRecipient.String != "" {
		recipient, err := entity.NewRecipient(rawRecipient.String)
		if err != nil {
			return entity.Handover{}, err
		}
		h.Recipient = recipient
	}
	return h, nil
}
