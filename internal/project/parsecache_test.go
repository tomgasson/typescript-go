package project

import (
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func TestParseCacheUsesSignatureForDiskFiles(t *testing.T) {
	cache := NewParseCache(RefCountCacheOptions{})
	fileName := "/src/module.ts"
	content := "export function add(a: number, b: number) { return a + b }\n"
	fh := newDiskFile(fileName, content)
	path := tspath.ToPath(fileName, "/", true)
	opts := ast.SourceFileParseOptions{FileName: fileName, Path: path, JSDocParsingMode: ast.JSDocParsingModeParseAll}
	key := NewParseCacheKey(opts, fh.Hash(), fh.Kind(), true)
	file := cache.Acquire(key, fh)
	t.Cleanup(func() { cache.Deref(key) })

	if !file.UseSignatureText() {
		t.Fatalf("expected signature text to be used for disk files")
	}
	if strings.Contains(file.Text(), "return a + b") {
		t.Fatalf("expected function body to be stripped in signature parse")
	}
}

func TestParseCacheFallsBackWhenNoSignature(t *testing.T) {
	cache := NewParseCache(RefCountCacheOptions{})
	fileName := "/src/noexports.ts"
	content := "const value = 1;\n"
	fh := newDiskFile(fileName, content)
	path := tspath.ToPath(fileName, "/", true)
	opts := ast.SourceFileParseOptions{FileName: fileName, Path: path, JSDocParsingMode: ast.JSDocParsingModeParseAll}
	key := NewParseCacheKey(opts, fh.Hash(), fh.Kind(), true)
	file := cache.Acquire(key, fh)
	t.Cleanup(func() { cache.Deref(key) })

	if !file.UseSignatureText() {
		t.Fatalf("expected signature path to be recorded")
	}
	if !strings.Contains(file.Text(), "const value") {
		t.Fatalf("expected original text when signature extraction produces nothing")
	}
}

func TestParseCacheSkipsSignatureForOverlays(t *testing.T) {
	cache := NewParseCache(RefCountCacheOptions{})
	fileName := "/src/overlay.tsx"
	content := "export const label = <div>overlay</div>;\n"
	overlay := newOverlay(fileName, content, 1, core.ScriptKindTSX)
	path := tspath.ToPath(fileName, "/", true)
	opts := ast.SourceFileParseOptions{FileName: fileName, Path: path, JSDocParsingMode: ast.JSDocParsingModeParseAll}
	useSignature := shouldUseSignatureText(opts, overlay.Kind(), overlay)
	if useSignature {
		t.Fatalf("expected overlay files to skip signature extraction")
	}
	key := NewParseCacheKey(opts, overlay.Hash(), overlay.Kind(), useSignature)
	file := cache.Acquire(key, overlay)
	t.Cleanup(func() { cache.Deref(key) })

	if file.UseSignatureText() {
		t.Fatalf("expected overlay parse to use full text")
	}
	if !strings.Contains(file.Text(), "overlay") {
		t.Fatalf("expected overlay content to remain intact")
	}
}
