package repo

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"

	"github.com/Akinori901/web-receipt/go/internal/entity"
	"github.com/go-pdf/fpdf"
)

// 日本語 TTF を実行ファイルに埋め込む。Lambda は外部 FS を読めないため同梱必須。
// フォントは SIL Open Font License（Noto Sans JP）。ライセンスは fonts/OFL.txt 参照。
//
//go:embed assets/fonts/NotoSansJP-Regular.ttf
var notoSansJP []byte

const pdfFontName = "NotoSansJP"

// ReceiptPDF は usecase.ReceiptPDFRenderer の実装。go-pdf/fpdf で定型領収書を描く。
type ReceiptPDF struct{}

func NewReceiptPDF() *ReceiptPDF { return &ReceiptPDF{} }

var taxLabels = map[string]string{
	"standard10": "標準税率 10%",
	"reduced8":   "軽減税率 8%",
	"exempt":     "非課税・対象外",
}

// Render は領収書を A4 縦の PDF バイト列に描画する。
func (ReceiptPDF) Render(_ context.Context, rc entity.Receipt) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	// 埋め込みフォントを登録（日本語のため UTF-8 フォント必須）。
	pdf.AddUTF8FontFromBytes(pdfFontName, "", notoSansJP)
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	// タイトル
	pdf.SetFont(pdfFontName, "", 24)
	pdf.CellFormat(0, 16, "領　収　書", "", 1, "C", false, 0, "")
	pdf.Ln(6)

	// 宛名
	pdf.SetFont(pdfFontName, "", 14)
	pdf.CellFormat(0, 10, rc.Recipient.String()+"　様", "B", 1, "L", false, 0, "")
	if addr := rc.RecipientAddress.String(); addr != "" {
		pdf.SetFont(pdfFontName, "", 10)
		pdf.CellFormat(0, 6, addr, "", 1, "L", false, 0, "")
	}
	pdf.Ln(6)

	// 金額（大きく強調）
	pdf.SetFont(pdfFontName, "", 20)
	pdf.CellFormat(0, 14, fmt.Sprintf("金 %s円", formatYen(rc.TotalAmount().Int64())), "B", 1, "C", false, 0, "")
	pdf.Ln(4)

	// 但し書き
	pdf.SetFont(pdfFontName, "", 11)
	proviso := rc.Proviso.String()
	if proviso == "" {
		proviso = "お品代として"
	}
	pdf.CellFormat(0, 8, "但し、"+proviso, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 8, "上記正に領収いたしました。", "", 1, "L", false, 0, "")
	pdf.Ln(6)

	// 内訳（税区分・税額）
	pdf.SetFont(pdfFontName, "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("税抜金額　%s円", formatYen(rc.NetAmount().Int64())), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("消費税額（%s）　%s円", taxLabel(rc.TaxCategory.String()), formatYen(rc.TaxAmount().Int64())), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	// 発行日・登録番号
	pdf.CellFormat(0, 6, "発行日　"+rc.IssuedOn.String(), "", 1, "L", false, 0, "")
	if !rc.RegistrationNumber.IsZero() {
		pdf.CellFormat(0, 6, "登録番号　"+rc.RegistrationNumber.String(), "", 1, "L", false, 0, "")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("PDF 生成に失敗: %w", err)
	}
	return buf.Bytes(), nil
}

func taxLabel(code string) string {
	if l, ok := taxLabels[code]; ok {
		return l
	}
	return code
}

// formatYen は金額を3桁区切りにする（12345 → 12,345）。
func formatYen(n int64) string {
	s := fmt.Sprintf("%d", n)
	neg := ""
	if n < 0 {
		neg = "-"
		s = s[1:]
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return neg + string(out)
}
