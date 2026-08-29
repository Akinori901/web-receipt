package entity_test

import (
	"testing"
	"time"

	"github.com/Akinori901/web-receipt/go/internal/entity"
)

func TestNewMoney(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		amount  int64
		wantErr bool
	}{
		{"正常", 11000, false},
		{"ゼロ", 0, false},
		{"負数は不正", -1, true},
		{"上限超過は不正", 2_000_000_000_000, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := entity.NewMoney(tt.amount)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewMoney(%d) err=%v, wantErr=%v", tt.amount, err, tt.wantErr)
			}
		})
	}
}

func TestNewTaxCategory(t *testing.T) {
	t.Parallel()
	for _, code := range []string{"standard10", "reduced8", "exempt"} {
		if _, err := entity.NewTaxCategory(code); err != nil {
			t.Errorf("NewTaxCategory(%q) が失敗: %v", code, err)
		}
	}
	if _, err := entity.NewTaxCategory("unknown"); err == nil {
		t.Error("未知の税区分が通ってしまった")
	}
}

func TestNewRegistrationNumber(t *testing.T) {
	t.Parallel()
	tests := []struct {
		raw     string
		wantErr bool
	}{
		{"", false},                // 未設定は許容
		{"T1234567890123", false},  // T + 13桁
		{"1234567890123", true},    // T が無い
		{"T123", true},             // 桁不足
	}
	for _, tt := range tests {
		_, err := entity.NewRegistrationNumber(tt.raw)
		if (err != nil) != tt.wantErr {
			t.Errorf("NewRegistrationNumber(%q) err=%v, wantErr=%v", tt.raw, err, tt.wantErr)
		}
	}
}

func TestNewRecipient(t *testing.T) {
	t.Parallel()
	if _, err := entity.NewRecipient("株式会社サンプル"); err != nil {
		t.Errorf("正常な宛名が失敗: %v", err)
	}
	if _, err := entity.NewRecipient("  "); err == nil {
		t.Error("空白のみの宛名が通ってしまった")
	}
}

// 標準税率10%の内税計算: 税込11000 → 税額1000・税抜10000。
func TestReceipt_TaxAmount_Standard10(t *testing.T) {
	t.Parallel()
	rc := buildReceipt(t, 11000, "standard10")
	if got := rc.TaxAmount().Int64(); got != 1000 {
		t.Errorf("TaxAmount = %d, want 1000", got)
	}
	if got := rc.NetAmount().Int64(); got != 10000 {
		t.Errorf("NetAmount = %d, want 10000", got)
	}
	if got := rc.TotalAmount().Int64(); got != 11000 {
		t.Errorf("TotalAmount = %d, want 11000", got)
	}
}

// 非課税は税額0。
func TestReceipt_TaxAmount_Exempt(t *testing.T) {
	t.Parallel()
	rc := buildReceipt(t, 5000, "exempt")
	if got := rc.TaxAmount().Int64(); got != 0 {
		t.Errorf("非課税の TaxAmount = %d, want 0", got)
	}
	if got := rc.NetAmount().Int64(); got != 5000 {
		t.Errorf("非課税の NetAmount = %d, want 5000", got)
	}
}

func buildReceipt(t *testing.T, amount int64, tax string) entity.Receipt {
	t.Helper()
	id, _ := entity.NewReceiptID("rid-1")
	owner, _ := entity.NewUserID("owner-1")
	recipient, _ := entity.NewRecipient("株式会社サンプル")
	address, _ := entity.NewRecipientAddress("")
	money, err := entity.NewMoney(amount)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	tc, err := entity.NewTaxCategory(tax)
	if err != nil {
		t.Fatalf("NewTaxCategory: %v", err)
	}
	reg, _ := entity.NewRegistrationNumber("")
	proviso, _ := entity.NewProviso("お品代として")
	issued, _ := entity.NewIssuedOn(time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC))
	return entity.NewReceipt(id, owner, recipient, address, money, tc, reg, proviso, issued)
}
