package usecase

import (
	"context"
	"time"

	"github.com/Akinori901/web-receipt/go/internal/entity"
)

// CreateReceiptInput は領収書作成の入力（プリミティブ）。
// 値オブジェクトへの変換は usecase 内で行い、controller には検証を漏らさない。
type CreateReceiptInput struct {
	Recipient          string
	RecipientAddress   string
	Amount             int64
	TaxCategory        string
	RegistrationNumber string
	Proviso            string
	IssuedOn           string // YYYY-MM-DD（空なら発行日=当日）
}

// ReceiptUseCase は領収書のユースケース。依存はすべて interface で注入する。
type ReceiptUseCase struct {
	receipts ReceiptRepo
	pdf      ReceiptPDFRenderer
	ids      IDGenerator
	clock    Clock
}

// NewReceiptUseCase は依存を注入して組み立てる。
func NewReceiptUseCase(receipts ReceiptRepo, pdf ReceiptPDFRenderer, ids IDGenerator, clock Clock) *ReceiptUseCase {
	return &ReceiptUseCase{receipts: receipts, pdf: pdf, ids: ids, clock: clock}
}

// Create は入力を検証して領収書を発行する。owner は認証済みユーザー。
func (uc *ReceiptUseCase) Create(ctx context.Context, owner entity.User, in CreateReceiptInput) (entity.Receipt, error) {
	recipient, err := entity.NewRecipient(in.Recipient)
	if err != nil {
		return entity.Receipt{}, err
	}
	address, err := entity.NewRecipientAddress(in.RecipientAddress)
	if err != nil {
		return entity.Receipt{}, err
	}
	amount, err := entity.NewMoney(in.Amount)
	if err != nil {
		return entity.Receipt{}, err
	}
	tax, err := entity.NewTaxCategory(in.TaxCategory)
	if err != nil {
		return entity.Receipt{}, err
	}
	registration, err := entity.NewRegistrationNumber(in.RegistrationNumber)
	if err != nil {
		return entity.Receipt{}, err
	}
	proviso, err := entity.NewProviso(in.Proviso)
	if err != nil {
		return entity.Receipt{}, err
	}
	issuedOn, err := uc.parseIssuedOn(in.IssuedOn)
	if err != nil {
		return entity.Receipt{}, err
	}
	id, err := entity.NewReceiptID(uc.ids.NewID())
	if err != nil {
		return entity.Receipt{}, err
	}

	receipt := entity.NewReceipt(id, owner.ID, recipient, address, amount, tax, registration, proviso, issuedOn)
	return uc.receipts.Save(ctx, receipt)
}

// Get は1件取得する。所有者以外はアクセスできない。
func (uc *ReceiptUseCase) Get(ctx context.Context, owner entity.User, id entity.ReceiptID) (entity.Receipt, error) {
	receipt, err := uc.receipts.FindByID(ctx, id)
	if err != nil {
		return entity.Receipt{}, err
	}
	if receipt.OwnerID != owner.ID {
		return entity.Receipt{}, entity.ErrReceiptForbidden
	}
	return receipt, nil
}

// List は所有者の領収書を絞り込んで返す。
func (uc *ReceiptUseCase) List(ctx context.Context, owner entity.User, filter entity.ReceiptFilter) ([]entity.Receipt, error) {
	return uc.receipts.ListByOwner(ctx, owner.ID, filter)
}

// Reissue は既存領収書を同一内容で再発行できるか確認して返す（内容は不変）。
func (uc *ReceiptUseCase) Reissue(ctx context.Context, owner entity.User, id entity.ReceiptID) (entity.Receipt, error) {
	receipt, err := uc.Get(ctx, owner, id)
	if err != nil {
		return entity.Receipt{}, err
	}
	if !receipt.CanReissue() {
		return entity.Receipt{}, entity.ErrReceiptForbidden
	}
	return receipt, nil
}

// GeneratePDF は領収書 PDF を生成して bytes を返す。所有者チェック込み。
func (uc *ReceiptUseCase) GeneratePDF(ctx context.Context, owner entity.User, id entity.ReceiptID) ([]byte, error) {
	receipt, err := uc.Get(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	return uc.pdf.Render(ctx, receipt)
}

// parseIssuedOn は入力日付文字列を IssuedOn に変換する。空なら当日。
func (uc *ReceiptUseCase) parseIssuedOn(raw string) (entity.IssuedOn, error) {
	if raw == "" {
		return entity.NewIssuedOn(uc.clock.Now())
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return entity.IssuedOn{}, entity.ErrInvalidIssuedOn
	}
	return entity.NewIssuedOn(t)
}
