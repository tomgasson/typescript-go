package project

import (
	"testing"
)

func TestExtractSignaturesSuccess(t *testing.T) {
	src := []byte(`export interface Foo { bar: string }
export function fn(a: number): void { return; }
`)

	result := extractSignatures(src)
	if !result.ok {
		t.Fatalf("expected extraction to succeed")
	}
	expected := "export interface Foo { bar: string }\nexport declare function fn(a: number): void;\n"
	if result.declarations != expected {
		t.Fatalf("unexpected declarations: %q", result.declarations)
	}
}

func TestExtractSignaturesUnsupportedExportFallsBack(t *testing.T) {
	src := []byte(`export const value = 1;`)

	result := extractSignatures(src)
	if result.ok {
		t.Fatalf("expected unsupported export to trigger fallback")
	}
	if result.declarations != "" {
		t.Fatalf("expected no declarations, got %q", result.declarations)
	}
}
