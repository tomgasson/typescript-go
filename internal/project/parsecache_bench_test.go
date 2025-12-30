package project

import (
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func BenchmarkParseCacheSignatureExtraction(b *testing.B) {
	const header = "export function heavy(x: number): number {"
	const body = " return x + 1; "
	const footer = "}\n"
	payload := header + strings.Repeat(body, 256) + footer

	diskFile := newDiskFile("/dev/null/file.ts", payload)
	compilerOptions := &core.CompilerOptions{}
	options := ast.SourceFileParseOptions{
		FileName:                       diskFile.FileName(),
		Path:                           tspath.ToPath(diskFile.FileName(), "", false),
		CompilerOptions:                compilerOptions.SourceFileAffecting(),
		ExternalModuleIndicatorOptions: ast.ExternalModuleIndicatorOptions{},
	}
	key := NewParseCacheKey(options, diskFile.Hash(), diskFile.Kind())

	b.Run("nonFocusedSignatureExtraction", func(b *testing.B) {
		cache := NewParseCache(RefCountCacheOptions{})
		for b.Loop() {
			file := cache.Acquire(key, diskFile)
			cache.Deref(key)
			if file == nil {
				b.Fatal("expected parsed source file")
			}
		}
	})

	overlay := newOverlay(diskFile.FileName(), payload, 1, diskFile.Kind())
	overlay.matchesDiskText = true
	overlayKey := NewParseCacheKey(options, overlay.Hash(), overlay.Kind())

	b.Run("focusedFullParse", func(b *testing.B) {
		cache := NewParseCache(RefCountCacheOptions{})
		for b.Loop() {
			file := cache.Acquire(overlayKey, overlay)
			cache.Deref(overlayKey)
			if file == nil {
				b.Fatal("expected parsed source file")
			}
		}
	})
}
