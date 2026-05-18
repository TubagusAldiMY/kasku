package email

import (
	"html/template"
	"strings"
	"testing"
)

func TestRenderTemplate_InjectsData(t *testing.T) {
	t.Parallel()
	tmpl := template.Must(template.New("greet.html").Parse(`Hello {{.Name}}!`))
	out, err := RenderTemplate(tmpl, "greet.html", map[string]interface{}{"Name": "Alice"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out != "Hello Alice!" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestRenderTemplate_HTMLEscapesUntrustedInput(t *testing.T) {
	t.Parallel()
	tmpl := template.Must(template.New("p.html").Parse(`<p>{{.Note}}</p>`))
	out, err := RenderTemplate(tmpl, "p.html", map[string]interface{}{
		"Note": `<script>alert("xss")</script>`,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if strings.Contains(out, "<script>") {
		t.Fatalf("html/template gagal escape script tag: %s", out)
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Fatalf("expected escaped script tag, got: %s", out)
	}
}

func TestRenderTemplate_ReturnsErrorForMissingTemplate(t *testing.T) {
	t.Parallel()
	tmpl := template.New("only.html")
	_, err := RenderTemplate(tmpl, "does-not-exist.html", nil)
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestNoOpSender_DoesNotError(t *testing.T) {
	t.Parallel()
	s := NewNoOpSender()
	if err := s.Send("a@b.com", "subj", "<p>body</p>"); err != nil {
		t.Fatalf("NoOpSender should never error, got: %v", err)
	}
}
