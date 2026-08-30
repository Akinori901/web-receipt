package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Akinori901/web-receipt/go/internal/entity"
	"github.com/Akinori901/web-receipt/go/internal/usecase"
)

// ── fake 群（DB も PDF ライブラリも無しでユースケースを検証する）──

type fakeReceipts struct {
	store map[string]entity.Receipt
}

func newFakeReceipts(seed ...entity.Receipt) *fakeReceipts {
	m := &fakeReceipts{store: map[string]entity.Receipt{}}
	for _, r := range seed {
		m.store[r.ID.String()] = r
	}
	return m
}

func (f *fakeReceipts) Save(_ context.Context, r entity.Receipt) (entity.Receipt, error) {
	f.store[r.ID.String()] = r
	return r, nil
}

func (f *fakeReceipts) FindByID(_ context.Context, id entity.ReceiptID) (entity.Receipt, error) {
	r, ok := f.store[id.String()]
	if !ok {
		return entity.Receipt{}, entity.ErrReceiptNotFound
	}
	return r, nil
}

func (f *fakeReceipts) ListByOwner(_ context.Context, owner entity.UserID, _ entity.ReceiptFilter) ([]entity.Receipt, error) {
	var out []entity.Receipt
	for _, r := range f.store {
		if r.OwnerID == owner {
			out = append(out, r)
		}
	}
	return out, nil
}

type fakePDF struct{}

func (fakePDF) Render(_ context.Context, _ entity.Receipt) ([]byte, error) {
	return []byte("%PDF-fake"), nil
}

type fixedID struct{ id string }

func (f fixedID) NewID() string { return f.id }

type fixedClock struct{ now time.Time }

func (f fixedClock) Now() time.Time { return f.now }

func newUC(repo usecase.ReceiptRepo) *usecase.ReceiptUseCase {
	return usecase.NewReceiptUseCase(
		repo, fakePDF{}, fixedID{id: "rid-1"},
		fixedClock{now: time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC)},
	)
}

func mustOwner(t *testing.T, id string) entity.User {
	t.Helper()
	uid, err := entity.NewUserID(id)
	if err != nil {
		t.Fatalf("NewUserID: %v", err)
	}
	email, _ := entity.NewEmail("owner@example.com")
	// テナントは所有者IDごとに別にする（別テナントのアクセス拒否を検証できるように）。
	tid, _ := entity.NewTenantID(id + "-tenant")
	u, err := entity.NewUserWithTenant(uid, email, tid, entity.AdminRole())
	if err != nil {
		t.Fatalf("NewUserWithTenant: %v", err)
	}
	return u
}

func TestReceiptUseCase_Create(t *testing.T) {
	t.Parallel()
	repo := newFakeReceipts()
	uc := newUC(repo)
	owner := mustOwner(t, "owner-1")

	rc, err := uc.Create(context.Background(), owner, usecase.CreateReceiptInput{
		Recipient:   "株式会社サンプル",
		Amount:      11000,
		TaxCategory: "standard10",
		IssuedOn:    "", // 空なら当日（fixedClock）
	})
	if err != nil {
		t.Fatalf("Create が失敗: %v", err)
	}
	if rc.ID.String() != "rid-1" {
		t.Errorf("ID = %q, want rid-1", rc.ID.String())
	}
	if rc.TaxAmount().Int64() != 1000 {
		t.Errorf("TaxAmount = %d, want 1000", rc.TaxAmount().Int64())
	}
	if rc.IssuedOn.String() != "2026-08-29" {
		t.Errorf("IssuedOn = %q, want 2026-08-29（当日）", rc.IssuedOn.String())
	}
}

func TestReceiptUseCase_Create_InvalidAmount(t *testing.T) {
	t.Parallel()
	uc := newUC(newFakeReceipts())
	owner := mustOwner(t, "owner-1")

	_, err := uc.Create(context.Background(), owner, usecase.CreateReceiptInput{
		Recipient:   "株式会社サンプル",
		Amount:      -1, // 不正
		TaxCategory: "standard10",
	})
	if !errors.Is(err, entity.ErrInvalidAmount) {
		t.Errorf("負の金額で err=%v, want ErrInvalidAmount", err)
	}
}

func TestReceiptUseCase_Get_Forbidden(t *testing.T) {
	t.Parallel()
	repo := newFakeReceipts()
	uc := newUC(repo)
	ownerA := mustOwner(t, "owner-A")
	ownerB := mustOwner(t, "owner-B")

	// A が作成
	rc, err := uc.Create(context.Background(), ownerA, usecase.CreateReceiptInput{
		Recipient: "宛名", Amount: 1000, TaxCategory: "exempt",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// B が A の領収書を取得 → 拒否
	_, err = uc.Get(context.Background(), ownerB, rc.ID)
	if !errors.Is(err, entity.ErrReceiptForbidden) {
		t.Errorf("他人の領収書取得で err=%v, want ErrReceiptForbidden", err)
	}

	// A 本人は取得できる
	if _, err := uc.Get(context.Background(), ownerA, rc.ID); err != nil {
		t.Errorf("本人の取得が失敗: %v", err)
	}
}

func TestReceiptUseCase_GeneratePDF(t *testing.T) {
	t.Parallel()
	repo := newFakeReceipts()
	uc := newUC(repo)
	owner := mustOwner(t, "owner-1")
	rc, _ := uc.Create(context.Background(), owner, usecase.CreateReceiptInput{
		Recipient: "宛名", Amount: 1000, TaxCategory: "exempt",
	})

	pdf, err := uc.GeneratePDF(context.Background(), owner, rc.ID)
	if err != nil {
		t.Fatalf("GeneratePDF: %v", err)
	}
	if len(pdf) == 0 {
		t.Error("PDF が空")
	}
}
