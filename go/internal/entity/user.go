// Package entity は最内層。ビジネスルールそのものを表す。
//
// **この層は他のどの内部パッケージにも依存しない。**
// 標準ライブラリ以外を import した時点で、それはもうエンティティではない。
// DB ドライバ・AWS SDK・HTTP フレームワークは、ここに来てはならない。
package entity

import (
	"errors"
	"regexp"
	"strings"
)

// ドメインのエラー。HTTP ステータスコードをここに持ち込まないこと。
// 「認証に失敗した」はドメインの語彙だが、「401」は controller の語彙である。
var (
	ErrInvalidEmail       = errors.New("メールアドレスの形式が不正です")
	ErrInvalidUserID      = errors.New("ユーザーIDが空です")
	ErrInvalidDisplayName = errors.New("表示名が不正です")
	ErrAuthFailed         = errors.New("メールアドレスまたはパスワードが正しくありません")
	ErrUserNotFound       = errors.New("ユーザーが見つかりません")
	ErrUserDeactivated    = errors.New("このアカウントは無効化されています")
)

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

const maxDisplayNameLength = 50

// Email は値オブジェクト。
//
// 生成時に検証することで「不正な Email が存在しない」ことを型で保証する。
// フィールドを非公開にして、後から書き換えられないようにしている。
type Email struct {
	value string
}

// NewEmail は検証済みの Email を返す。
func NewEmail(raw string) (Email, error) {
	if !emailPattern.MatchString(raw) {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: raw}, nil
}

// NewOptionalEmail は空を許す Email を返す（発行者の代表メール等の任意項目用）。
// 空でなければ形式を検証する。
func NewOptionalEmail(raw string) (Email, error) {
	if raw == "" {
		return Email{}, nil
	}
	return NewEmail(raw)
}

// IsZero は Email が未設定かを返す。
func (e Email) IsZero() bool { return e.value == "" }

func (e Email) String() string { return e.value }

// LocalPart は @ より前を返す。既定の表示名の導出に使う。
func (e Email) LocalPart() string {
	if i := strings.Index(e.value, "@"); i >= 0 {
		return e.value[:i]
	}
	return e.value
}

// UserID は値オブジェクト。Cognito の sub をそのまま識別子として扱う。
//
// 素の string を持ち回すと「どの ID なのか」が型から失われるため包む。
type UserID struct {
	value string
}

func NewUserID(raw string) (UserID, error) {
	if strings.TrimSpace(raw) == "" {
		return UserID{}, ErrInvalidUserID
	}
	return UserID{value: raw}, nil
}

func (u UserID) String() string { return u.value }

// DisplayName は値オブジェクト。
type DisplayName struct {
	value string
}

func NewDisplayName(raw string) (DisplayName, error) {
	if strings.TrimSpace(raw) == "" || len([]rune(raw)) > maxDisplayNameLength {
		return DisplayName{}, ErrInvalidDisplayName
	}
	return DisplayName{value: raw}, nil
}

func (d DisplayName) String() string { return d.value }

// User はエンティティ。同一性を持つ（値が変わっても ID が同じなら同じ User）。
//
// TenantID は所属組織、Role は組織内の役割（admin/member）。
// 領収書の認可・組織設定の可否はこの2つで判定する。
type User struct {
	ID          UserID
	Email       Email
	DisplayName DisplayName
	IsActive    bool
	TenantID    TenantID
	Role        Role
}

// NewUserWithTenant は組織所属と役割つきでユーザーを組み立てる。
// 表示名はメールアドレスから導出する。招待フロー・初回サインインで使う。
func NewUserWithTenant(id UserID, email Email, tenantID TenantID, role Role) (User, error) {
	name, err := NewDisplayName(email.LocalPart())
	if err != nil {
		return User{}, err
	}
	return User{ID: id, Email: email, DisplayName: name, IsActive: true, TenantID: tenantID, Role: role}, nil
}

// CanSignIn はサインイン可能かを判定する（ビジネスルール）。
//
// この判定を usecase や controller の if で書かないこと。
// ルールをエンティティに置かないと、同じ判定が各所へ散らばる。
func (u User) CanSignIn() bool { return u.IsActive }

// IsAdmin は組織の管理者かを判定する（ビジネスルール）。
// 組織設定の変更・member 登録の可否判定に使う。
func (u User) IsAdmin() bool { return u.Role.IsAdmin() }

// Deactivate はアカウントを無効化する。
func (u *User) Deactivate() { u.IsActive = false }

// Equals はエンティティの等価性。識別子のみで決まる。
func (u User) Equals(other User) bool { return u.ID == other.ID }
