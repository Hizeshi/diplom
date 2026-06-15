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

//go:embed logo.png
var logoPNG []byte

const (
	fontFamily = "DejaVu"
	pageWidth  = 210.0
	margin     = 14.0
	contentW   = pageWidth - 2*margin

	// Brand colours
	colorHeaderR, colorHeaderG, colorHeaderB = 14, 14, 35    // deep navy
	colorAccentR, colorAccentG, colorAccentB = 108, 99, 255  // indigo
	colorRowEvenR, colorRowEvenG, colorRowEvenB = 245, 244, 255
	colorTotalR, colorTotalG, colorTotalB = 232, 230, 255
)

// Generate builds a PDF for the given quote and returns the raw bytes.
func Generate(req quote.Request) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(margin, margin, margin)
	pdf.SetAutoPageBreak(true, margin+10)

	pdf.AddUTF8FontFromBytes(fontFamily, "", fontRegular)
	pdf.AddUTF8FontFromBytes(fontFamily, "B", fontBold)

	pdf.AddPage()

	drawHeader(pdf)
	pdf.Ln(6)

	if req.CompanyName != "" || req.ContactName != "" || req.Phone != "" || req.Email != "" {
		drawClientBlock(pdf, req)
		pdf.Ln(5)
	}

	drawItemsTable(pdf, req.Items)

	if req.Note != "" {
		pdf.Ln(5)
		drawNote(pdf, req.Note)
	}

	drawFooter(pdf)

	var w bytes.Buffer
	if err := pdf.Output(&w); err != nil {
		return nil, fmt.Errorf("pdf: output: %w", err)
	}
	return w.Bytes(), nil
}

// ─── Header ──────────────────────────────────────────────────────────────────

func drawHeader(pdf *gofpdf.Fpdf) {
	headerH := 28.0

	// Dark background band
	pdf.SetFillColor(colorHeaderR, colorHeaderG, colorHeaderB)
	pdf.Rect(0, 0, pageWidth, headerH, "F")

	// Logo
	if len(logoPNG) > 0 {
		opt := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
		pdf.RegisterImageOptionsReader("logo", opt, bytes.NewReader(logoPNG))
		logoSize := 18.0
		pdf.ImageOptions("logo", margin, (headerH-logoSize)/2, logoSize, logoSize, false, opt, 0, "")
	}

	// Brand name
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(fontFamily, "B", 18)
	pdf.SetXY(margin+21, 6)
	pdf.CellFormat(50, 8, "L-XOR", "", 0, "L", false, 0, "")

	// Tagline
	pdf.SetFont(fontFamily, "", 7)
	pdf.SetTextColor(180, 178, 220)
	pdf.SetXY(margin+21, 15)
	pdf.CellFormat(60, 5, "Электрофурнитура JASMART", "", 0, "L", false, 0, "")

	// Document title — right side
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(fontFamily, "B", 11)
	pdf.SetXY(0, 8)
	pdf.CellFormat(pageWidth-margin, 7, "КОММЕРЧЕСКОЕ ПРЕДЛОЖЕНИЕ", "", 1, "R", false, 0, "")

	// Date — right side
	pdf.SetFont(fontFamily, "", 8)
	pdf.SetTextColor(180, 178, 220)
	pdf.SetXY(0, 16)
	pdf.CellFormat(pageWidth-margin, 5, time.Now().Format("02.01.2006"), "", 1, "R", false, 0, "")

	// Accent underline
	pdf.SetFillColor(colorAccentR, colorAccentG, colorAccentB)
	pdf.Rect(0, headerH-1.5, pageWidth, 1.5, "F")

	pdf.SetY(headerH + 1)
	pdf.SetTextColor(0, 0, 0)
}

// ─── Client block ─────────────────────────────────────────────────────────────

func drawClientBlock(pdf *gofpdf.Fpdf, req quote.Request) {
	pdf.SetFont(fontFamily, "B", 9)
	pdf.SetTextColor(colorAccentR, colorAccentG, colorAccentB)
	pdf.CellFormat(contentW, 5, "ПОЛУЧАТЕЛЬ", "", 1, "L", false, 0, "")

	// Thin accent line under label
	pdf.SetFillColor(colorAccentR, colorAccentG, colorAccentB)
	pdf.Rect(margin, pdf.GetY(), 22, 0.4, "F")
	pdf.Ln(3)

	pdf.SetTextColor(0, 0, 0)
	fieldPairs := [][2]string{
		{"Компания", req.CompanyName},
		{"Контакт", req.ContactName},
		{"Телефон", req.Phone},
		{"Email", req.Email},
	}
	for _, p := range fieldPairs {
		if p[1] == "" {
			continue
		}
		pdf.SetFont(fontFamily, "B", 9)
		pdf.CellFormat(28, 5.5, p[0]+":", "", 0, "L", false, 0, "")
		pdf.SetFont(fontFamily, "", 9)
		pdf.CellFormat(contentW-28, 5.5, p[1], "", 1, "L", false, 0, "")
	}
}

// ─── Items table ─────────────────────────────────────────────────────────────

func drawItemsTable(pdf *gofpdf.Fpdf, items []quote.Item) {
	colW := [4]float64{94, 28, 22, 38}
	headers := [4]string{"Наименование товара", "Артикул", "Кол-во", "Сумма, ₸"}

	// Section label
	pdf.SetFont(fontFamily, "B", 9)
	pdf.SetTextColor(colorAccentR, colorAccentG, colorAccentB)
	pdf.CellFormat(contentW, 5, "СОСТАВ КП", "", 1, "L", false, 0, "")
	pdf.SetFillColor(colorAccentR, colorAccentG, colorAccentB)
	pdf.Rect(margin, pdf.GetY(), 20, 0.4, "F")
	pdf.Ln(3)

	// Table header
	pdf.SetFillColor(colorHeaderR, colorHeaderG, colorHeaderB)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(fontFamily, "B", 9)
	for i, h := range headers {
		align := "L"
		if i > 1 {
			align = "C"
		}
		pdf.CellFormat(colW[i], 7.5, h, "", 0, align, true, 0, "")
	}
	pdf.Ln(-1)

	// Table rows
	pdf.SetFont(fontFamily, "", 8.5)
	var total float64
	for idx, item := range items {
		if idx%2 == 0 {
			pdf.SetFillColor(colorRowEvenR, colorRowEvenG, colorRowEvenB)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.SetTextColor(20, 20, 40)
		sum := float64(item.Quantity) * item.Price
		total += sum

		// Name cell may be long — use MultiCell trick for wrapping
		x := pdf.GetX()
		y := pdf.GetY()

		// Calculate row height based on how many lines the name wraps to
		lines := pdf.SplitLines([]byte(item.Name), colW[0]-2)
		lineH := 5.0
		rowH := float64(len(lines)) * lineH
		if rowH < 7.0 {
			rowH = 7.0
		}

		// Fill row background
		pdf.Rect(margin, y, contentW, rowH, "F")

		// Name (wrapping)
		pdf.SetXY(x, y)
		pdf.MultiCell(colW[0], lineH, item.Name, "", "L", false)

		// Other cells aligned to row top, full row height
		pdf.SetXY(x+colW[0], y)
		pdf.CellFormat(colW[1], rowH, item.Article, "", 0, "C", false, 0, "")
		pdf.CellFormat(colW[2], rowH, fmt.Sprintf("%d", item.Quantity), "", 0, "C", false, 0, "")
		pdf.CellFormat(colW[3], rowH, formatMoney(sum), "", 0, "R", false, 0, "")

		pdf.SetXY(margin, y+rowH)
	}

	// Separator line
	pdf.SetDrawColor(colorAccentR, colorAccentG, colorAccentB)
	pdf.SetLineWidth(0.3)
	pdf.Line(margin, pdf.GetY(), margin+contentW, pdf.GetY())

	// Total row
	pdf.SetFillColor(colorTotalR, colorTotalG, colorTotalB)
	pdf.SetTextColor(colorHeaderR, colorHeaderG, colorHeaderB)
	pdf.SetFont(fontFamily, "B", 10)
	totalLabelW := colW[0] + colW[1] + colW[2]
	pdf.CellFormat(totalLabelW, 9, "ИТОГО:", "", 0, "R", true, 0, "")
	pdf.SetFont(fontFamily, "B", 11)
	pdf.SetTextColor(colorAccentR, colorAccentG, colorAccentB)
	pdf.CellFormat(colW[3], 9, formatMoney(total), "", 1, "R", true, 0, "")
}

// ─── Note ────────────────────────────────────────────────────────────────────

func drawNote(pdf *gofpdf.Fpdf, note string) {
	pdf.SetFont(fontFamily, "B", 9)
	pdf.SetTextColor(colorAccentR, colorAccentG, colorAccentB)
	pdf.CellFormat(contentW, 5, "ПРИМЕЧАНИЕ", "", 1, "L", false, 0, "")
	pdf.SetFillColor(colorAccentR, colorAccentG, colorAccentB)
	pdf.Rect(margin, pdf.GetY(), 26, 0.4, "F")
	pdf.Ln(3)

	pdf.SetFont(fontFamily, "", 9)
	pdf.SetTextColor(40, 40, 60)
	pdf.SetFillColor(245, 244, 255)
	pdf.MultiCell(contentW, 5, note, "", "L", true)
}

// ─── Footer ──────────────────────────────────────────────────────────────────

func drawFooter(pdf *gofpdf.Fpdf) {
	// Disable auto page break so footer cells don't spill to a new page.
	pdf.SetAutoPageBreak(false, 0)
	defer pdf.SetAutoPageBreak(true, margin+10)

	footerY := 279.0 // 297 - 18mm from bottom
	lineH := 5.0

	// Accent separator line
	pdf.SetFillColor(colorAccentR, colorAccentG, colorAccentB)
	pdf.Rect(0, footerY, pageWidth, 0.5, "F")

	// Dark background band
	pdf.SetFillColor(colorHeaderR, colorHeaderG, colorHeaderB)
	pdf.Rect(0, footerY+0.5, pageWidth, 17.5, "F")

	// First row: brand name + website
	pdf.SetXY(margin, footerY+3)
	pdf.SetFont(fontFamily, "B", 9)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(contentW/2, lineH, "L-XOR", "", 0, "L", false, 0, "")
	pdf.SetFont(fontFamily, "", 8)
	pdf.SetTextColor(180, 178, 220)
	pdf.CellFormat(contentW/2, lineH, "iq-home.kz", "", 1, "R", false, 0, "")

	// Second row: disclaimer
	pdf.SetXY(margin, footerY+3+lineH+1)
	pdf.SetFont(fontFamily, "", 7.5)
	pdf.SetTextColor(140, 138, 180)
	pdf.CellFormat(contentW, 4, "Данное КП сформировано автоматически и действительно в течение 7 дней", "", 0, "C", false, 0, "")
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func formatMoney(v float64) string {
	return fmt.Sprintf("%.0f ₸", v)
}
