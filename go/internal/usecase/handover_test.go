package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Akinori901/web-receipt/go/internal/entity"
	"github.com/Akinori901/web-receipt/go/internal/usecase"
)

// fakeHandovers は usecase.HandoverRepo の in-memory fake（viewID キー）。
type fakeHandovers struct {
	byReceipt map[string]entity.Handover
	byView    map[string]string // viewID -> receiptID
}

func newFakeHandovers() *fakeHandovers {
	return &fakeHandovers{byReceipt: map[string]entity.Handover{}, byView: map[string]string{}}
}

func (f *fakeHandovers) Save(_ context.Context, h entity.Handover) (entity.Handover, error) {
	f.byReceipt[h.ReceiptID.String()] = h
	f.byView[h.ViewID.String()] = h.ReceiptID.String()
	return h, nil
}

func (f *fakeHandovers) FindByReceiptID(_ context.Context, id entity.ReceiptID) (entity.Handover, error) {
	h, ok := f.byReceipt[id.String()]
	if !ok {
		return entity.Handover{}, entity.ErrHandoverNotFound
	}
	return h, nil
}

func (f *fakeHandovers) FindByViewID(_ context.Context, viewID entity.ViewID) (entity.Handover, error) {
	rid, ok := f.byView[viewID.String()]
	if !ok {
		return entity.Handover{}, entity.ErrHandoverNotFound
	}
	return f.byReceipt[rid], nil
}

func (f *fakeHandovers) IncrementURLView(_ context.Context, viewID entity.ViewID) error {
	rid := f.byView[viewID.String()]
	h := f.byReceipt[rid]
	h.URLViewCount++
	f.byReceipt[rid] = h
	return nil
}

func (f *fakeHandovers) IncrementPDF(_ context.Context, viewID entity.ViewID) error {
	rid := f.byView[viewID.String()]
	h := f.byReceipt[rid]
	h.PDFCount++
	f.byReceipt[rid] = h
	return nil
}

// seqID は Issue のたびに異なる viewID/password を返す（決定的な連番）。
type seqID struct{ n *int }

func (s seqID) NewID() string {
	*s.n++
	return "id-" + string(rune('0'+*s.n))
}

func newHandoverUC(receipts usecase.ReceiptRepo, tenants usecase.TenantRepo) (*usecase.HandoverUseCase, *fakeHandovers) {
	h := newFakeHandovers()
	n := 0
	uc := usecase.NewHandoverUseCase(h, receipts, tenants, fakePDF{}, seqID{n: &n})
	return uc, h
}

// setupReceipt は owner の領収書を1件作って返す（handover テストの前提）。
func setupReceipt(t *testing.T) (entity.User, entity.Receipt, *fakeReceipts, *fakeTenants) {
	t.Helper()
	owner := mustOwner(t, "owner-1")
	tname, _ := entity.NewCorporateName("発行者")
	tenant := entity.NewTenant(owner.TenantID, tname, entity.Address{}, entity.Phone{}, entity.Email{}, entity.RegistrationNumber{})
	tenants := newFakeTenants()
	_, _ = tenants.Save(context.Background(), tenant)

	receipts := newFakeReceipts()
	rid, _ := entity.NewReceiptID("rid-1")
	rc := entity.NewReceipt(
		rid, owner.TenantID, owner.ID,
		mustRecipient(t, "宛名"), entity.RecipientAddress{},
		mustMoney(t, 1000), mustTax(t, "exempt"),
		entity.RegistrationNumber{}, entity.Proviso{},
		mustIssuedOn(t),
	)
	_, _ = receipts.Save(context.Background(), rc)
	return owner, rc, receipts, tenants
}

func TestHandoverUseCase_IssueAndReceiverFlow(t *testing.T) {
	t.Parallel()
	owner, rc, receipts, tenants := setupReceipt(t)
	uc, _ := newHandoverUC(receipts, tenants)

	// 発行側が閲覧ID/PWを発行する。
	status, err := uc.Issue(context.Background(), owner, rc.ID)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	viewID, _ := entity.NewViewID(status.ViewID)
	pw := status.ViewPassword

	// 受領側が閲覧（URL閲覧数 +1）。
	if _, err := uc.View(context.Background(), viewID, pw); err != nil {
		t.Fatalf("View: %v", err)
	}
	// 受領側が PDF 出力（PDF出力数 +1）。
	if _, err := uc.OutputPDF(context.Background(), viewID, pw); err != nil {
		t.Fatalf("OutputPDF: %v", err)
	}
	// パスワード誤りは拒否。
	if _, err := uc.View(context.Background(), viewID, "wrong"); !errors.Is(err, entity.ErrHandoverAuthFailed) {
		t.Errorf("誤PWで err=%v, want ErrHandoverAuthFailed", err)
	}

	// 発行側 Status はカウントを加算しない。
	s2, err := uc.Status(context.Background(), owner, rc.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if s2.URLViewCount != 1 {
		t.Errorf("URLViewCount = %d, want 1（発行側閲覧で増えない）", s2.URLViewCount)
	}
	if s2.PDFCount != 1 {
		t.Errorf("PDFCount = %d, want 1", s2.PDFCount)
	}
}

func TestHandoverUseCase_ConfirmRecipientOnlyOnce(t *testing.T) {
	t.Parallel()
	owner, rc, receipts, tenants := setupReceipt(t)
	uc, _ := newHandoverUC(receipts, tenants)
	status, _ := uc.Issue(context.Background(), owner, rc.ID)
	viewID, _ := entity.NewViewID(status.ViewID)
	pw := status.ViewPassword

	// 1回目の宛名確定は成功。
	if err := uc.ConfirmRecipient(context.Background(), viewID, pw, "確定された宛名"); err != nil {
		t.Fatalf("1回目 ConfirmRecipient: %v", err)
	}
	// 2回目は拒否される。
	if err := uc.ConfirmRecipient(context.Background(), viewID, pw, "別の宛名"); !errors.Is(err, entity.ErrRecipientAlreadyConfirmed) {
		t.Errorf("2回目 err=%v, want ErrRecipientAlreadyConfirmed", err)
	}
}

// ── テストヘルパ ──

func mustRecipient(t *testing.T, s string) entity.Recipient {
	t.Helper()
	r, err := entity.NewRecipient(s)
	if err != nil {
		t.Fatalf("NewRecipient: %v", err)
	}
	return r
}

func mustMoney(t *testing.T, n int64) entity.Money {
	t.Helper()
	m, err := entity.NewMoney(n)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	return m
}

func mustTax(t *testing.T, c string) entity.TaxCategory {
	t.Helper()
	tc, err := entity.NewTaxCategory(c)
	if err != nil {
		t.Fatalf("NewTaxCategory: %v", err)
	}
	return tc
}

func mustIssuedOn(t *testing.T) entity.IssuedOn {
	t.Helper()
	io, err := entity.NewIssuedOn(time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewIssuedOn: %v", err)
	}
	return io
}
