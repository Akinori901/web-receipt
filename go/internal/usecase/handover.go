package usecase

import (
	"context"

	"github.com/Akinori901/web-receipt/go/internal/entity"
)

// HandoverStatus は発行側に返す受け渡し状況。
type HandoverStatus struct {
	ViewID             string
	ViewPassword       string
	URLViewCount       int
	PDFCount           int
	RecipientConfirmed bool
}

// HandoverUseCase は領収書の受け渡し管理ユースケース。
//
// 発行側(認証)は Issue/Status を使い、カウントを加算しない。
// 受領側(非認証)は View/OutputPDF/ConfirmRecipient を使い、カウントを加算する。
type HandoverUseCase struct {
	handovers HandoverRepo
	receipts  ReceiptRepo
	tenants   TenantRepo
	pdf       ReceiptPDFRenderer
	ids       IDGenerator
}

func NewHandoverUseCase(
	handovers HandoverRepo,
	receipts ReceiptRepo,
	tenants TenantRepo,
	pdf ReceiptPDFRenderer,
	ids IDGenerator,
) *HandoverUseCase {
	return &HandoverUseCase{handovers: handovers, receipts: receipts, tenants: tenants, pdf: pdf, ids: ids}
}

// ── 発行側（認証・カウント非加算）──

// Issue は領収書に閲覧ID/PWを発行する。既に発行済みなら既存を返す（冪等）。
func (uc *HandoverUseCase) Issue(ctx context.Context, user entity.User, receiptID entity.ReceiptID) (HandoverStatus, error) {
	// 領収書が自組織のものか確認する。
	receipt, err := uc.receipts.FindByID(ctx, receiptID)
	if err != nil {
		return HandoverStatus{}, err
	}
	if receipt.TenantID != user.TenantID {
		return HandoverStatus{}, entity.ErrReceiptForbidden
	}

	// 既に発行済みならそれを返す。
	if existing, err := uc.handovers.FindByReceiptID(ctx, receiptID); err == nil {
		return toHandoverStatus(existing), nil
	}

	viewID, err := entity.NewViewID(uc.ids.NewID())
	if err != nil {
		return HandoverStatus{}, err
	}
	password, err := entity.NewViewPassword(uc.ids.NewID())
	if err != nil {
		return HandoverStatus{}, err
	}
	h := entity.NewHandover(receiptID, viewID, password)
	saved, err := uc.handovers.Save(ctx, h)
	if err != nil {
		return HandoverStatus{}, err
	}
	return toHandoverStatus(saved), nil
}

// Status は受け渡し状況を返す（発行側・カウント非加算）。
func (uc *HandoverUseCase) Status(ctx context.Context, user entity.User, receiptID entity.ReceiptID) (HandoverStatus, error) {
	receipt, err := uc.receipts.FindByID(ctx, receiptID)
	if err != nil {
		return HandoverStatus{}, err
	}
	if receipt.TenantID != user.TenantID {
		return HandoverStatus{}, entity.ErrReceiptForbidden
	}
	h, err := uc.handovers.FindByReceiptID(ctx, receiptID)
	if err != nil {
		return HandoverStatus{}, err
	}
	return toHandoverStatus(h), nil
}

// ── 受領側（非認証・カウント加算）──

// ReceiptView は受領側に返す領収書の内容。
type ReceiptView struct {
	Recipient          string
	Amount             int64
	TaxCategory        string
	IssuedOn           string
	Proviso            string
	RecipientConfirmed bool
}

// View は受領側が領収書を閲覧する。パスワード照合し URL 閲覧数を加算する。
func (uc *HandoverUseCase) View(ctx context.Context, viewID entity.ViewID, password string) (ReceiptView, error) {
	h, receipt, err := uc.authenticate(ctx, viewID, password)
	if err != nil {
		return ReceiptView{}, err
	}
	if err := uc.handovers.IncrementURLView(ctx, viewID); err != nil {
		return ReceiptView{}, err
	}
	// 宛名は確定済みならその値を優先する。
	recipient := receipt.Recipient.String()
	if h.RecipientConfirmed {
		recipient = h.Recipient.String()
	}
	return ReceiptView{
		Recipient:          recipient,
		Amount:             receipt.Amount.Int64(),
		TaxCategory:        receipt.TaxCategory.String(),
		IssuedOn:           receipt.IssuedOn.String(),
		Proviso:            receipt.Proviso.String(),
		RecipientConfirmed: h.RecipientConfirmed,
	}, nil
}

// OutputPDF は受領側が PDF を出力する。パスワード照合し PDF 出力数を加算する。
func (uc *HandoverUseCase) OutputPDF(ctx context.Context, viewID entity.ViewID, password string) ([]byte, error) {
	h, receipt, err := uc.authenticate(ctx, viewID, password)
	if err != nil {
		return nil, err
	}
	// 宛名確定済みなら領収書の宛名を確定値で差し替える（PDF に反映）。
	if h.RecipientConfirmed {
		receipt.Recipient = h.Recipient
	}
	issuer, err := uc.tenants.FindByID(ctx, receipt.TenantID)
	if err != nil {
		return nil, err
	}
	pdf, err := uc.pdf.Render(ctx, receipt, issuer)
	if err != nil {
		return nil, err
	}
	if err := uc.handovers.IncrementPDF(ctx, viewID); err != nil {
		return nil, err
	}
	return pdf, nil
}

// ConfirmRecipient は受領側が宛名を1回だけ確定する。
func (uc *HandoverUseCase) ConfirmRecipient(ctx context.Context, viewID entity.ViewID, password, name string) error {
	h, _, err := uc.authenticate(ctx, viewID, password)
	if err != nil {
		return err
	}
	recipient, err := entity.NewRecipient(name)
	if err != nil {
		return err
	}
	if err := h.ConfirmRecipient(recipient); err != nil {
		return err // 2回目以降は ErrRecipientAlreadyConfirmed
	}
	_, err = uc.handovers.Save(ctx, h)
	return err
}

// authenticate は viewID/password を照合し、handover と対応する領収書を返す。
func (uc *HandoverUseCase) authenticate(ctx context.Context, viewID entity.ViewID, password string) (entity.Handover, entity.Receipt, error) {
	h, err := uc.handovers.FindByViewID(ctx, viewID)
	if err != nil {
		return entity.Handover{}, entity.Receipt{}, err
	}
	if !h.VerifyPassword(password) {
		return entity.Handover{}, entity.Receipt{}, entity.ErrHandoverAuthFailed
	}
	receipt, err := uc.receipts.FindByID(ctx, h.ReceiptID)
	if err != nil {
		return entity.Handover{}, entity.Receipt{}, err
	}
	return h, receipt, nil
}

func toHandoverStatus(h entity.Handover) HandoverStatus {
	return HandoverStatus{
		ViewID:             h.ViewID.String(),
		ViewPassword:       h.ViewPassword.String(),
		URLViewCount:       h.URLViewCount,
		PDFCount:           h.PDFCount,
		RecipientConfirmed: h.RecipientConfirmed,
	}
}
