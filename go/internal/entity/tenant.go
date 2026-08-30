// Package entity は最内層。標準ライブラリ以外を import しないこと。
package entity

import (
	"errors"
	"strings"
)

// テナント（組織）ドメインのエラー。HTTP ステータスは持ち込まない。
var (
	ErrInvalidTenantID      = errors.New("組織IDが空です")
	ErrInvalidCorporateName = errors.New("法人名が不正です")
	ErrInvalidAddress       = errors.New("住所が不正です")
	ErrInvalidPhone         = errors.New("電話番号が不正です")
	ErrInvalidRole          = errors.New("役割が不正です")
	ErrTenantNotFound       = errors.New("組織が見つかりません")
	ErrTenantForbidden      = errors.New("この組織を操作する権限がありません")
)

const (
	maxCorporateNameLength = 200
	maxTenantAddressLength = 500
	maxPhoneLength         = 30
)

// TenantID は値オブジェクト（UUID 文字列を想定）。
type TenantID struct{ value string }

func NewTenantID(raw string) (TenantID, error) {
	if strings.TrimSpace(raw) == "" {
		return TenantID{}, ErrInvalidTenantID
	}
	return TenantID{value: raw}, nil
}

func (t TenantID) String() string { return t.value }

// CorporateName は発行者の法人名（値オブジェクト・必須）。
type CorporateName struct{ value string }

func NewCorporateName(raw string) (CorporateName, error) {
	s := strings.TrimSpace(raw)
	if s == "" || len([]rune(s)) > maxCorporateNameLength {
		return CorporateName{}, ErrInvalidCorporateName
	}
	return CorporateName{value: s}, nil
}

func (c CorporateName) String() string { return c.value }

// Address は発行者の住所（値オブジェクト・任意）。
type Address struct{ value string }

func NewAddress(raw string) (Address, error) {
	if len([]rune(raw)) > maxTenantAddressLength {
		return Address{}, ErrInvalidAddress
	}
	return Address{value: raw}, nil
}

func (a Address) String() string { return a.value }

// Phone は発行者の代表電話番号（値オブジェクト・任意）。
type Phone struct{ value string }

func NewPhone(raw string) (Phone, error) {
	if len([]rune(raw)) > maxPhoneLength {
		return Phone{}, ErrInvalidPhone
	}
	return Phone{value: raw}, nil
}

func (p Phone) String() string { return p.value }

// Role はアカウントの役割（値オブジェクト・enum）。
type Role struct{ code string }

const (
	roleAdmin  = "admin"  // 組織を管理し member を登録できる
	roleMember = "member" // 領収書の発行・閲覧はできるが組織設定は不可
)

// NewRole は許可された役割のみを受け付ける。
func NewRole(code string) (Role, error) {
	switch code {
	case roleAdmin, roleMember:
		return Role{code: code}, nil
	default:
		return Role{}, ErrInvalidRole
	}
}

func (r Role) String() string { return r.code }
func (r Role) IsAdmin() bool  { return r.code == roleAdmin }

// AdminRole / MemberRole は生成のヘルパ（検証済みの値を返す）。
func AdminRole() Role  { return Role{code: roleAdmin} }
func MemberRole() Role { return Role{code: roleMember} }

// Tenant はエンティティ（組織 = 発行者情報の器）。同一性は ID で決まる。
//
// 領収書の発行者情報（法人名・住所・電話・メール・インボイス登録番号）を1つ持つ。
// 組織メンバー全員がこの発行者情報を共有し、組織の領収書履歴を閲覧できる。
type Tenant struct {
	ID                 TenantID
	CorporateName      CorporateName
	Address            Address
	Phone              Phone
	Email              Email              // 代表メール（既存の Email VO を再利用）
	RegistrationNumber RegistrationNumber // インボイス登録番号（既存 VO を再利用）
}

// NewTenant は検証済みの各値オブジェクトから組織を組み立てるファクトリ。
func NewTenant(
	id TenantID,
	name CorporateName,
	address Address,
	phone Phone,
	email Email,
	registration RegistrationNumber,
) Tenant {
	return Tenant{
		ID:                 id,
		CorporateName:      name,
		Address:            address,
		Phone:              phone,
		Email:              email,
		RegistrationNumber: registration,
	}
}

// Equals はエンティティの等価性。識別子のみで決まる。
func (t Tenant) Equals(other Tenant) bool { return t.ID == other.ID }
