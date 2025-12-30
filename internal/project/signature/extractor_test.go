package signature

import (
	"strconv"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func TestExtractReturnsSignature(t *testing.T) {
	src := []byte("export function foo(x: number): number { return x }\n")
	sig, ok := Extract(src)
	if !ok {
		t.Fatalf("expected signature extraction to succeed")
	}
	expected := "export declare function foo(x: number): number;\n"
	if sig != expected {
		t.Fatalf("unexpected signature:\n%s", sig)
	}
}

func TestExtractWithoutExportsFails(t *testing.T) {
	if _, ok := Extract([]byte("const value = 1;\n")); ok {
		t.Fatalf("expected extraction to fail without exports")
	}
}

func BenchmarkExtractVsParse(b *testing.B) {
	source := buildLargeSource(2000)
	sourceBytes := []byte(source)
	fileName := tspath.GetNormalizedAbsolutePath("bench.ts", "/")
	opts := ast.SourceFileParseOptions{FileName: fileName, Path: tspath.Path(fileName)}

	b.Run("full-parse", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			parser.ParseSourceFile(opts, source, core.ScriptKindTS)
		}
	})

	b.Run("signature-only", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if sig, ok := Extract(sourceBytes); ok {
				parser.ParseSourceFile(opts, sig, core.ScriptKindTS)
			} else {
				b.Fatalf("signature extraction failed")
			}
		}
	})
}

func buildLargeSource(count int) string {
	var b strings.Builder
	b.Grow(count * 64)
	for i := 0; i < count; i++ {
		b.WriteString("export function fn")
		b.WriteString(strconv.Itoa(i))
		b.WriteString("(value: number): number {\n")
		b.WriteString("  const doubled = value * 2;\n")
		b.WriteString("  const squared = value * value;\n")
		b.WriteString("  return doubled + squared;\n")
		b.WriteString("}\n")
	}
	return b.String()
}
