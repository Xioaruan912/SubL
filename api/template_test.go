package api

import "testing"

func TestValidateTemplateTextAndOutline(t *testing.T) {
	outline, errs := validateTemplateText("clash.yaml", "port: 7890\nproxy-groups:\n  - name: Auto\nproxies: {{ nodes }}\n")
	if len(errs) != 0 {
		t.Fatalf("expected valid template, got %v", errs)
	}
	if len(outline) != 3 || outline[1].Key != "proxy-groups" {
		t.Fatalf("unexpected outline: %#v", outline)
	}
}

func TestValidateTemplateTextReportsYAMLError(t *testing.T) {
	_, errs := validateTemplateText("bad.yaml", "proxies: [\n")
	if len(errs) == 0 {
		t.Fatal("expected yaml syntax error")
	}
}
