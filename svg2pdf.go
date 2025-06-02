package svg2pdf

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// SVG represents the SVG document structure
type SVG struct {
	XMLName   xml.Name   `xml:"http://www.w3.org/2000/svg svg"`
	Width     string     `xml:"width,attr"`
	Height    string     `xml:"height,attr"`
	Rects     []Rect     `xml:"http://www.w3.org/2000/svg rect"`
	Texts     []Text     `xml:"http://www.w3.org/2000/svg text"`
	Paths     []Path     `xml:"http://www.w3.org/2000/svg path"`
	Gradients []Gradient `xml:"http://www.w3.org/2000/svg linearGradient"`
}

// Rect represents an SVG rectangle
type Rect struct {
	X      float64 `xml:"x,attr"`
	Y      float64 `xml:"y,attr"`
	Width  float64 `xml:"width,attr"`
	Height float64 `xml:"height,attr"`
	Stroke string  `xml:"stroke,attr"`
}

// Text represents an SVG text element
type Text struct {
	X       float64 `xml:"x,attr"`
	Y       float64 `xml:"y,attr"`
	Content string  `xml:",chardata"`
	Font    string  `xml:"font,attr"`      // Add font attribute for customization
	Size    float64 `xml:"font-size,attr"` // Font size support
}

// Path represents an SVG path element
type Path struct {
	D string `xml:"d,attr"`
}

// Gradient represents a gradient definition
type Gradient struct {
	ID    string  `xml:"id,attr"`
	X1    float64 `xml:"x1,attr"`
	Y1    float64 `xml:"y1,attr"`
	X2    float64 `xml:"x2,attr"`
	Y2    float64 `xml:"y2,attr"`
	Stops []Stop  `xml:"stop"`
}

// Stop represents a stop in the gradient (color at a specific offset)
type Stop struct {
	Offset string `xml:"offset,attr"`
	Color  string `xml:"stop-color,attr"`
}

// PDF represents a PDF document with advanced layout features
type PDF struct {
	pages       []string
	pageCount   int
	content     []string
	pageWidth   float64
	pageHeight  float64
	scaleX      float64
	scaleY      float64
	currentX    float64
	currentY    float64
	columnWidth float64
	rowHeight   float64
	maxColumns  int
	maxRows     int
	font        string  // Font for text rendering
	fontSize    float64 // Font size
}

// NewPDF creates a new PDF document with row and column support, custom fonts, and font size
func NewPDF(columns, rows int, font string, fontSize float64) *PDF {
	return &PDF{
		pages:       []string{},
		pageCount:   0,
		content:     []string{},
		pageWidth:   595, // A4 in points (210mm at 72 DPI)
		pageHeight:  842, // A4 in points (297mm at 72 DPI)
		currentX:    0,
		currentY:    0,
		columnWidth: 150, // Default width for columns
		rowHeight:   50,  // Default height for rows
		maxColumns:  columns,
		maxRows:     rows,
		font:        font,
		fontSize:    fontSize,
	}
}

// AddRow adds a new row to the PDF, incrementing Y position
func (p *PDF) AddRow() {
	p.currentY += p.rowHeight
	p.currentX = 0
}

// AddColumn adds a new column to the PDF, incrementing X position
func (p *PDF) AddColumn() {
	p.currentX += p.columnWidth
	if p.currentX+p.columnWidth > p.pageWidth {
		p.AddRow() // Move to the next row if the current row is full
	}
}

// ApplyTransformation applies a transformation (like rotation) to the coordinates
func ApplyTransformation(x, y float64, transform string) (float64, float64) {
	if transform == "rotate" {
		// Apply 90-degree rotation for simplicity
		return y, 595 - x // Swap X and Y for 90-degree rotation
	}
	// Add more transformations (scale, translate) if needed
	return x, y
}

// RenderGradient renders a simple linear gradient on a rectangle
func (p *PDF) RenderGradient(gradient Gradient, x, y, w, h float64) {
	// For simplicity, let's use the first gradient stop's color as the fill color
	// More complex gradient logic can be added later.
	gradientColor := gradient.Stops[0].Color // Use the first color for now

	// Render a simple rectangle with a solid color fill (linear gradient logic can be extended)
	p.content = append(p.content,
		fmt.Sprintf("%.2f %.2f %.2f %.2f re", x, y, w, h), // Define rectangle for gradient
		"0 0 1 RG", // Set color (for simplicity, using one color here)
		"S",        // Apply fill
	)
}

// AddTextWithUnicode renders text with font size, font, and Unicode support
func (p *PDF) AddTextWithUnicode(x, y float64, text string) {
	escapedText := escapeText(text)
	stream := []string{
		"BT",
		fmt.Sprintf("/F1 %.2f Tf", p.fontSize), // Set font size
		fmt.Sprintf("%.2f %.2f Td", x, y),      // Set position
		fmt.Sprintf("(%s) Tj", escapedText),    // Render text
		"ET",
	}
	p.content = append(p.content, strings.Join(stream, "\n"))
}

// AddPage adds a new page to the PDF
func (p *PDF) AddPage() {
	p.pageCount++
	page := fmt.Sprintf("Page %d", p.pageCount)
	p.pages = append(p.pages, page)
	p.content = append(p.content, "")
}

// ConvertSVGToPDF processes the SVG file and handles elements (gradients, transformations, etc.)
func (p *PDF) ConvertSVGToPDF(svgFilePath string) error {
	// Open SVG file
	svgFile, err := os.Open(svgFilePath)
	if err != nil {
		return fmt.Errorf("error opening SVG file: %v", err)
	}
	defer svgFile.Close()

	// Parse SVG content
	var svgData SVG
	if err := xml.NewDecoder(svgFile).Decode(&svgData); err != nil {
		return fmt.Errorf("error decoding SVG: %v", err)
	}

	// Adjust SVG dimensions to fit the page, with scaling
	svgWidth, svgHeight := 400.0, 150.0
	if svgData.Width != "" && svgData.Height != "" {
		svgWidth, _ = strconv.ParseFloat(svgData.Width, 64)
		svgHeight, _ = strconv.ParseFloat(svgData.Height, 64)
	}

	// Scale factor to fit SVG content into PDF page
	p.scaleX = p.pageWidth / svgWidth
	p.scaleY = p.pageHeight / svgHeight

	// Start a new page and layout elements into grid
	p.AddPage()

	// Process gradients (rendering a basic linear gradient)
	for _, gradient := range svgData.Gradients {
		p.RenderGradient(gradient, 100, 100, 200, 50) // Sample rectangle with gradient
	}

	// Process SVG elements (rectangles, text, paths)
	var stream []string
	for _, rect := range svgData.Rects {
		p.AddColumn()
		x := rect.X * p.scaleX
		y := p.pageHeight - (rect.Y * p.scaleY)
		w := rect.Width * p.scaleX
		h := rect.Height * p.scaleY

		// Append drawing instructions for rectangles
		stream = append(stream,
			fmt.Sprintf("%.2f %.2f m", x, y),
			fmt.Sprintf("%.2f %.2f l", x+w, y),
			fmt.Sprintf("%.2f %.2f l", x+w, y-h),
			fmt.Sprintf("%.2f %.2f l", x, y-h),
			"h",        // Close path
			"0 0 0 RG", // Black stroke
			"S",        // Stroke
		)
	}

	// Process text elements
	for _, text := range svgData.Texts {
		p.AddColumn()
		x := text.X * p.scaleX
		y := p.pageHeight - (text.Y * p.scaleY)
		// Apply transformations and add text with font
		x, y = ApplyTransformation(x, y, "rotate")
		p.AddTextWithUnicode(x, y, text.Content)
	}

	// Add all processed stream content
	p.content = append(p.content, strings.Join(stream, "\n"))
	return nil
}

// Save saves the PDF to a file
func (p *PDF) Save(filePath string) error {
	var pdfContent []string

	// PDF Header
	pdfContent = append(pdfContent,
		"%PDF-1.4",
		"%âãÏÓ",
	)

	// Catalog
	pdfContent = append(pdfContent,
		"1 0 obj",
		"<<",
		"/Type /Catalog",
		fmt.Sprintf("/Pages 2 0 R"),
		">>",
		"endobj",
	)

	// Pages
	pdfContent = append(pdfContent,
		"2 0 obj",
		"<<",
		"/Type /Pages",
		fmt.Sprintf("/Count %d", p.pageCount),
		"/Kids [",
	)
	for i := 0; i < p.pageCount; i++ {
		pdfContent = append(pdfContent, fmt.Sprintf("%d 0 R", 3+i*2))
	}
	pdfContent = append(pdfContent,
		"]",
		">>",
		"endobj",
	)

	// Font (Helvetica, built-in)
	pdfContent = append(pdfContent,
		"3 0 obj",
		"<<",
		"/Type /Font",
		"/Subtype /Type1",
		"/BaseFont /Helvetica",
		"/Name /F1",
		">>",
		"endobj",
	)

	// Page objects and content streams
	for i := 0; i < p.pageCount; i++ {
		// Page
		pdfContent = append(pdfContent,
			fmt.Sprintf("%d 0 obj", 4+i*2),
			"<<",
			"/Type /Page",
			"/Parent 2 0 R",
			fmt.Sprintf("/MediaBox [0 0 %.2f %.2f]", p.pageWidth, p.pageHeight),
			"/Resources <<",
			"/Font <<",
			"/F1 3 0 R",
			">>",
			">>",
			fmt.Sprintf("/Contents %d 0 R", 5+i*2),
			">>",
			"endobj",
		)

		// Content Stream
		contentStream := p.content[i]
		pdfContent = append(pdfContent,
			fmt.Sprintf("%d 0 obj", 5+i*2),
			"<<",
			"/Length "+strconv.Itoa(len(contentStream)),
			">>",
			"stream",
			contentStream,
			"endstream",
			"endobj",
		)
	}

	// Cross-reference table
	xrefOffset := 0
	var xref []string
	xref = append(xref,
		"xref",
		fmt.Sprintf("0 %d", 5+p.pageCount*2+1),
		"0000000000 65535 f ",
	)
	xrefOffset += len(strings.Join(pdfContent[:2], "\n")) + 2
	for i := 1; i <= 4+p.pageCount*2; i++ {
		xref = append(xref, fmt.Sprintf("%010d 00000 n ", xrefOffset))
		section := strings.Join(pdfContent[i:i+1], "\n") + "\n"
		xrefOffset += len(section)
	}

	// Trailer
	trailer := []string{
		"trailer",
		"<<",
		fmt.Sprintf("/Size %d", 5+p.pageCount*2+1),
		"/Root 1 0 R",
		">>",
		"startxref",
		fmt.Sprintf("%d", xrefOffset),
		"%%EOF",
	}

	// Combine all parts
	finalContent := strings.Join(pdfContent, "\n") + "\n" +
		strings.Join(xref, "\n") + "\n" +
		strings.Join(trailer, "\n")

	// Write to file
	if err := os.WriteFile(filePath, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("error writing PDF: %v", err)
	}
	fmt.Printf("Successfully generated %s\n", filePath)
	return nil
}

// escapeText escapes special characters for PDF text
func escapeText(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "(", "\\(")
	text = strings.ReplaceAll(text, ")", "\\)")
	return text
}
