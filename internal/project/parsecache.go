package project

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/zeebo/xxh3"
)

type ParseCacheKey struct {
	ast.SourceFileParseOptions
	ScriptKind       core.ScriptKind
	Hash             xxh3.Uint128
	UseSignatureText bool
}

func NewParseCacheKey(
	options ast.SourceFileParseOptions,
	hash xxh3.Uint128,
	scriptKind core.ScriptKind,
	useSignatureText bool,
) ParseCacheKey {
	return ParseCacheKey{
		SourceFileParseOptions: options,
		Hash:                   hash,
		ScriptKind:             scriptKind,
		UseSignatureText:       useSignatureText,
	}
}

type ParseCache = RefCountCache[ParseCacheKey, *ast.SourceFile, FileHandle]

func NewParseCache(options RefCountCacheOptions) *ParseCache {
	return NewRefCountCache(
		options,
		func(key ParseCacheKey, fh FileHandle) *ast.SourceFile {
			useSignatureText := key.UseSignatureText && shouldUseSignatureText(key.SourceFileParseOptions, key.ScriptKind, fh)
			text := fh.Content()
			if useSignatureText {
				if signature, ok := extractModuleSignature([]byte(text)); ok {
					file := parser.ParseSourceFile(key.SourceFileParseOptions, signature, key.ScriptKind)
					if len(file.Diagnostics()) == 0 {
						file.Hash = fh.Hash()
						file.SetUseSignatureText(true)
						return file
					}
				}
			}

			file := parser.ParseSourceFile(key.SourceFileParseOptions, text, key.ScriptKind)
			file.Hash = fh.Hash()
			file.SetUseSignatureText(useSignatureText)
			return file
		},
		nil,
	)
}

func shouldUseSignatureText(opts ast.SourceFileParseOptions, scriptKind core.ScriptKind, fh FileHandle) bool {
	if fh.IsOverlay() {
		return false
	}

	if scriptKind == core.ScriptKindJSON {
		return false
	}

	if tspath.IsDeclarationFileName(opts.FileName) {
		return false
	}

	return true
}
