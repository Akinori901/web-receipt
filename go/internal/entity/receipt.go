// Package entity は最内層。標準ライブラリ以外を import しないこと。
package entity

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// 領収書ドメインのエラー。HTTP ステータスは持ち込まない（controller の語彙）。
var (
	ErrInvalidReceiptID          = errors.New("領収書IDが空です")
	ErrInvalidAmount             = errors.New("金額が不正です")
	ErrInvalidRecipient          = errors.New("宛名が不正です")
	ErrInvalidRecipientAddress   = errors.New("宛先住所が不正です")
	ErrInvalidTaxCategory        = errors.New("税区分が不正です")
	ErrInvalidRegistrationNumber = errors.New("登録番号の形式が不正です")
	ErrInvalidProviso            = errors.New("但し書きが不正です")
	ErrInvalidIssuedOn           = errors.New("発行日が不正です")
	ErrReceiptNotFound           = errors.New("領収書が見つかりません")
	ErrReceiptForbidden          = errors.New("この領収書を操作する権限がありません")
)

const (
	maxRecipientLength = 200
	maxAddressLength   = 500
	maxProvisoLength   = 200
	maxReceiptAmount   = 1_000_000_000_000 // 1兆円。桁あふれ・入力ミスの上限
)

// ReceiptID は値オブジェクト（UUID 文字列を想定）。
type ReceiptID struct{ value string }

func NewReceiptID(raw string) (ReceiptID, error) {
	if strings.TrimSpace(raw) == "" {
		return ReceiptID{}, ErrInvalidReceiptID
	}
	return ReceiptID{value: raw}, nil
}

func (r ReceiptID) String() string { return r.value }

// Money は値オブジェクト。最小通貨単位（円）を int64 で持つ。
// 金額に float を使わない（丸め誤差が金額を狂わせる）。
type Money struct{ amount int64 }

func NewMoney(amount int64) (Money, error) {
	if amount < 0 || amount > maxReceiptAmount {
		return Money{}, ErrInvalidAmount
	}
	return Money{amount: amount}, nil
}

func (m Money) Int64() int64 { return m.amount }

// TaxCategory は消費税の区分（値オブジェクト・enum）。
type TaxCategory struct{ code string }

const (
	taxStandard10 = "standard10" // 標準税率 10%
	taxReduced8   = "reduced8"   // 軽減税率 8%
	taxExempt     = "exempt"     // 非課税・対象外
)

// NewTaxCategory は許可された区分のみを受け付ける。
func NewTaxCategory(code string) (TaxCategory, error) {
	switch code {
	case taxStandard10, taxReduced8, taxExempt:
		return TaxCategory{code: code}, nil
	default:
		return TaxCategory{}, ErrInvalidTaxCategory
	}
}

func (t TaxCategory) String() string { return t.code }

// ratePermille は税率（千分率）。内税計算に使う。
func (t TaxCategory) ratePermille() int64 {
	switch t.code {
	case taxStandard10:
		return 100
	case taxReduced8:
		return 80
	default:
		return 0
	}
}

var registrationPattern = regexp.MustCompile(`^T\d{13}$`)

// RegistrationNumber はインボイス登録番号（T + 13桁）。任意項目なので空を許す。
type RegistrationNumber struct{ value string }

func NewRegistrationNumber(raw string) (RegistrationNumber, error) {
	if raw == "" {
		return RegistrationNumber{}, nil // 未設定は許容
	}
	if !registrationPattern.MatchString(raw) {
		return RegistrationNumber{}, ErrInvalidRegistrationNumber
	}
	return RegistrationNumber{value: raw}, nil
}

func (r RegistrationNumber) String() string { return r.value }
func (r RegistrationNumber) IsZero() bool   { return r.value == "" }

// Recipient は領収書の宛名（値オブジェクト・必須）。
type Recipient struct{ value string }

func NewRecipient(raw string) (Recipient, error) {
	s := strings.TrimSpace(raw)
	if s == "" || len([]rune(s)) > maxRecipientLength {
		return Recipient{}, ErrInvalidRecipient
	}
	return Recipient{value: s}, nil
}

func (r Recipient) String() string { return r.value }

// RecipientAddress は宛先住所（値オブジェクト・任意）。
type RecipientAddress struct{ value string }

func NewRecipientAddress(raw string) (RecipientAddress, error) {
	if len([]rune(raw)) > maxAddressLength {
		return RecipientAddress{}, ErrInvalidRecipientAddress
	}
	return RecipientAddress{value: raw}, nil
}

func (r RecipientAddress) String() string { return r.value }

// Proviso は但し書き（値オブジェクト・任意）。
type Proviso struct{ value string }

func NewProviso(raw string) (Proviso, error) {
	if len([]rune(raw)) > maxProvisoLength {
		return Proviso{}, ErrInvalidProviso
	}
	return Proviso{value: raw}, nil
}

func (p Proviso) String() string { return p.value }

// IssuedOn は発行日（値オブジェクト・日付のみ）。
type IssuedOn struct{ value time.Time }

// NewIssuedOn は発行日を組み立てる。時刻は落として日付だけを保持する。
func NewIssuedOn(t time.Time) (IssuedOn, error) {
	if t.IsZero() {
		return IssuedOn{}, ErrInvalidIssuedOn
	}
	y, m, d := t.Date()
	return IssuedOn{value: time.Date(y, m, d, 0, 0, 0, 0, time.UTC)}, nil
}

func (i IssuedOn) Time() time.Time { return i.value }
func (i IssuedOn) String() string  { return i.value.Format("2006-01-02") }

// Receipt はエンティティ。同一性は ID で決まる。
//
// 宛名・住所は「値」で直接保持する（顧客マスタへの参照 ID を持たない）。
// 領収書は発行時点のスナップショットが正であり、後で顧客名が変わっても
// 発行済みの宛名は変わってはならない。また領収書サービスを顧客管理サービスに
// 依存させず単体で完結させるため、ここでは値としてコピー保持する。
type Receipt struct {
	ID                 ReceiptID
	TenantID           TenantID // 所属組織。認可はこれで判定する
	OwnerID            UserID   // 発行者個人。「誰が発行したか」の履歴表示用
	Recipient          Recipient
	RecipientAddress   RecipientAddress
	Amount             Money
	TaxCategory        TaxCategory
	RegistrationNumber RegistrationNumber
	Proviso            Proviso
	IssuedOn           IssuedOn
	CreatedAt          time.Time
}

// NewReceipt は検証済みの各値オブジェクトから領収書を組み立てるファクトリ。
func NewReceipt(
	id ReceiptID,
	tenant TenantID,
	owner UserID,
	recipient Recipient,
	address RecipientAddress,
	amount Money,
	tax TaxCategory,
	registration RegistrationNumber,
	proviso Proviso,
	issuedOn IssuedOn,
) Receipt {
	return Receipt{
		ID:                 id,
		TenantID:           tenant,
		OwnerID:            owner,
		Recipient:          recipient,
		RecipientAddress:   address,
		Amount:             amount,
		TaxCategory:        tax,
		RegistrationNumber: registration,
		Proviso:            proviso,
		IssuedOn:           issuedOn,
	}
}

// TaxAmount は税込金額（Amount）に含まれる消費税額を返す（内税計算）。
//
// この計算を usecase / controller の if で書かないこと。
// 税額 = 税込 × 率 / (100 + 率)。非課税は 0。
func (r Receipt) TaxAmount() Money {
	rate := r.TaxCategory.ratePermille() // 千分率（10%→100）
	if rate == 0 {
		return Money{amount: 0}
	}
	// amount * rate / (1000 + rate)。int64 のまま丸める（切り捨て）。
	tax := r.Amount.amount * rate / (1000 + rate)
	return Money{amount: tax}
}

// NetAmount は税抜金額（税込 − 税額）を返す。
func (r Receipt) NetAmount() Money {
	return Money{amount: r.Amount.amount - r.TaxAmount().amount}
}

// TotalAmount は請求総額（＝税込金額）を返す。
func (r Receipt) TotalAmount() Money { return r.Amount }

// CanReissue は再発行可能かを判定する（ビジネスルール）。
// 発行済みの領収書はいつでも同一内容で再発行してよい。
func (r Receipt) CanReissue() bool { return true }

// Equals はエンティティの等価性。識別子のみで決まる。
func (r Receipt) Equals(other Receipt) bool { return r.ID == other.ID }

// ReceiptFilter は領収書一覧の絞り込み条件。ゼロ値は「条件なし」。
//
// repo と usecase の両方が使う型なので entity に置く（どちらかに置くと層が逆流する）。
type ReceiptFilter struct {
	Recipient string // 宛名の部分一致（空=絞らない）
	IssuedGTE string // 発行日の下限 YYYY-MM-DD（空=絞らない）
	IssuedLTE string // 発行日の上限 YYYY-MM-DD（空=絞らない）
}
