package project

import (
	"fmt"
	"strings"
	testing "testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func BenchmarkSignatureExtraction(b *testing.B) {
	sourceText := buildSignatureBenchmarkSource()
	fileName := "/src/bench.ts"
	path := tspath.ToPath(fileName, "/", true)
	options := ast.SourceFileParseOptions{FileName: fileName, Path: path, JSDocParsingMode: ast.JSDocParsingModeParseAll}
	scriptKind := core.GetScriptKindFromFileName(fileName)

	b.Run("full-parse", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			parser.ParseSourceFile(options, sourceText, scriptKind)
		}
	})

	b.Run("signature-fast-path", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			signatureText, ok := extractModuleSignature([]byte(sourceText))
			if !ok {
				b.Fatal("expected signature to be extracted")
			}
			parser.ParseSourceFile(options, signatureText, scriptKind)
		}
	})
}

func buildSignatureBenchmarkSource() string {
	var b strings.Builder
	b.WriteString("export interface Box { value: number }\n")
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&b, "export function fn%d(input: number): number { return input + %d; }\n", i, i)
	}
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&b, "export class Item%d { constructor(readonly v: number) {} }\n", i)
	}
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&b, "export type Alias%d = Box;\n", i)
	}
	return b.String()
}
