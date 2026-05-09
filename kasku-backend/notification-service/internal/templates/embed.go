// Package templates menyediakan akses ke HTML email templates yang di-embed ke dalam binary.
// Templates di-embed saat compile time sehingga tidak ada dependency pada filesystem saat runtime.
package templates

import (
	"embed"
	"html/template"
)

//go:embed *.html
var templateFS embed.FS

// LoadEmailTemplates mem-parse semua HTML template dari embed filesystem.
// Mengembalikan error jika ada template yang tidak valid secara sintaks.
func LoadEmailTemplates() (*template.Template, error) {
	return template.ParseFS(templateFS, "*.html")
}
