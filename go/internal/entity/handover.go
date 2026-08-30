// Package entity は最内層。標準ライブラリ以外を import しないこと。
package entity

import (
	"errors"
	"strings"
)

// 受け渡し（handover）ドメインのエラー。HTTP ステータスは持ち込まない。
var (
	ErrInvalidViewID             = errors.New("閲覧IDが空です")
	ErrInvalidViewPassword       = errors.New("閲覧パスワードが空です")
	ErrHandoverNotFound          = errors.New("受け渡しが見つかりません")
	ErrHandoverAuthFailed        = errors.New("閲覧IDまたはパスワードが正しくありません")
	ErrRecipientAlreadyConfirmed = errors.New("宛名は既に確定済みで変更できません")
)

// ViewID は受領側が確認画面にアクセスするための公開ID（値オブジェクト）。
type ViewID struct{ value string }

func NewViewID(raw string) (ViewID, error) {
	if strings.TrimSpace(raw) == "" {
		return ViewID{}, ErrInvalidViewID
	}
	return ViewID{value: raw}, nil
}

func (v ViewID) String() string { return v.value }

// ViewPassword は受領側の閲覧パスワード（値オブジェクト・平文保存でよい要件）。
type ViewPassword struct{ value string }

func NewViewPassword(raw string) (ViewPassword, error) {
	if strings.TrimSpace(raw) == "" {
		return ViewPassword{}, ErrInvalidViewPassword
	}
	return ViewPassword{value: raw}, nil
}

func (p ViewPassword) String() string { return p.value }

// Handover は領収書の受け渡し管理（アクセス状況・宛名確定）。
//
// 領収書ごとに1つ。発行側が viewID/viewPassword を発行して受領側へ連絡し、
// 受領側は非認証でアクセスして PDF 出力・宛名確定を行う。
// アクセス状況（URL閲覧数/PDF出力数）を記録する（発行側の閲覧はカウントに含めない）。
type Handover struct {
	ReceiptID          ReceiptID
	ViewID             ViewID
	ViewPassword       ViewPassword
	URLViewCount       int
	PDFCount           int
	RecipientConfirmed bool
	Recipient          Recipient // 受領側が確定した宛名。未確定はゼロ値。
}

// NewHandover は発行時の受け渡しを組み立てる（カウント0・宛名未確定）。
func NewHandover(receiptID ReceiptID, viewID ViewID, password ViewPassword) Handover {
	return Handover{
		ReceiptID:    receiptID,
		ViewID:       viewID,
		ViewPassword: password,
	}
}

// VerifyPassword は閲覧パスワードを照合する（平文比較）。
func (h Handover) VerifyPassword(raw string) bool { return h.ViewPassword.value == raw }

// ConfirmRecipient は受領側が宛名を確定する。2回目以降は変更できない（ビジネスルール）。
func (h *Handover) ConfirmRecipient(recipient Recipient) error {
	if h.RecipientConfirmed {
		return ErrRecipientAlreadyConfirmed
	}
	h.Recipient = recipient
	h.RecipientConfirmed = true
	return nil
}
