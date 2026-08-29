package repo

import (
	"crypto/rand"
	"fmt"
	"time"
)

// UUIDGenerator は usecase.IDGenerator の実装（UUID v4 相当）。
// 外部ライブラリを足さず crypto/rand で生成する。
type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator { return &UUIDGenerator{} }

// NewID は UUID v4 形式の文字列を返す。
func (UUIDGenerator) NewID() string {
	var b [16]byte
	// crypto/rand は失敗しない前提だが、万一に備えゼロ値でも一意性は id 側で担保される。
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// SystemClock は usecase.Clock の実装。
type SystemClock struct{}

func NewSystemClock() *SystemClock { return &SystemClock{} }

func (SystemClock) Now() time.Time { return time.Now() }
