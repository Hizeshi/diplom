package pdf

import (
	"bytes"
	_ "embed"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"

	"iq-home/backend/internal/domain/quote"
)

//go:embed fonts/DejaVuSans.ttf
var fontRegular []byte

//go:embed fonts/DejaVuSans-Bold.ttf
var fontBold []byte

const (
	fontFamily = "DejaVu"
	pageWidth  = 210.0
	margin     = 15.0
	contentW   = pageWidth - 2*margin
)

// Generate builds a PDF for the given quote and returns the raw bytes.
func Generate(req quote.Request) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(margin, margin, margin)
	pdf.SetAutoPageBreak(true, margin)

	// Register UTF-8 fonts from embedded bytes
	pdf.AddUTF8FontFromBytes(fontFamily, "", fontRegular)
	pdf.AddUTF8FontFromBytes(fontFamily, "B", fontBold)

	pdf.AddPage()

	// ── Header ──────────────────────────────────────────────────────────────
	pdf.SetFont(fontFamily, "B", 20)
	pdf.CellFormat(contentW, 10, "Коммерческое предложение", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont(fontFamily, "", 10)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(contentW, 6, fmt.Sprintf("Дата: %s", time.Now().Format("02.01.2006")), "", 1, "R", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(2)

	// ── Client info ─────────────────────────────────────────────────────────
	if req.CompanyName != "" || req.ContactName != "" || req.Phone != "" || req.Email != "" {
		pdf.SetFont(fontFamily, "B", 11)
		pdf.CellFormat(contentW, 7, "Клиент", "", 1, "", false, 0, "")
		pdf.SetFont(fontFamily, "", 10)

		writeField(pdf, "Компания", req.CompanyName)
		writeField(pdf, "Контакт", req.ContactName)
		writeField(pdf, "Телефон", req.Phone)
		writeField(pdf, "Email", req.Email)
		pdf.Ln(4)
	}

	// ── Items table ─────────────────────────────────────────────────────────
	colW := [4]float64{80, 30, 25, 45} // Наименование | Артикул | Кол-во | Сумма

	// Table header
	pdf.SetFillColor(41, 128, 185)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(fontFamily, "B", 10)
	headers := [4]string{"Наименование", "Артикул", "Кол-во", "Сумма"}
	for i, h := range headers {
		pdf.CellFormat(colW[i], 8, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Table rows
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont(fontFamily, "", 9)
	var total float64
	for idx, item := range req.Items {
		if idx%2 == 0 {
			pdf.SetFillColor(240, 248, 255)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		sum := float64(item.Quantity) * item.Price
		total += sum

		pdf.CellFormat(colW[0], 7, item.Name, "1", 0, "L", true, 0, "")
		pdf.CellFormat(colW[1], 7, item.Article, "1", 0, "C", true, 0, "")
		pdf.CellFormat(colW[2], 7, fmt.Sprintf("%d", item.Quantity), "1", 0, "C", true, 0, "")
		pdf.CellFormat(colW[3], 7, formatMoney(sum), "1", 0, "R", true, 0, "")
		pdf.Ln(-1)
	}

	// Total row
	pdf.SetFillColor(220, 237, 200)
	pdf.SetFont(fontFamily, "B", 10)
	pdf.CellFormat(colW[0]+colW[1]+colW[2], 8, "ИТОГО", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colW[3], 8, formatMoney(total), "1", 0, "R", true, 0, "")
	pdf.Ln(-1)

	// ── Note ────────────────────────────────────────────────────────────────
	if req.Note != "" {
		pdf.Ln(6)
		pdf.SetFont(fontFamily, "B", 10)
		pdf.CellFormat(contentW, 6, "Примечание", "", 1, "", false, 0, "")
		pdf.SetFont(fontFamily, "", 9)
		pdf.MultiCell(contentW, 5, req.Note, "1", "L", false)
	}

	// ── Footer ──────────────────────────────────────────────────────────────
	pdf.SetY(-20)
	pdf.SetFont(fontFamily, "", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.CellFormat(contentW, 5, "IQ-Home | iq-home.kz", "", 0, "C", false, 0, "")

	var w bytes.Buffer
	if err := pdf.Output(&w); err != nil {
		return nil, fmt.Errorf("pdf: output: %w", err)
	}
	return w.Bytes(), nil
}

func writeField(pdf *gofpdf.Fpdf, label, value string) {
	if value == "" {
		return
	}
	pdf.SetFont(fontFamily, "B", 10)
	pdf.CellFormat(40, 6, label+":", "", 0, "", false, 0, "")
	pdf.SetFont(fontFamily, "", 10)
	pdf.CellFormat(contentW-40, 6, value, "", 1, "", false, 0, "")
}

func formatMoney(v float64) string {
	return fmt.Sprintf("%.0f ₸", v)
}
