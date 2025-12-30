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
	ScriptKind core.ScriptKind
	Hash       xxh3.Uint128
}

func NewParseCacheKey(
	options ast.SourceFileParseOptions,
	hash xxh3.Uint128,
	scriptKind core.ScriptKind,
) ParseCacheKey {
	return ParseCacheKey{
		SourceFileParseOptions: options,
		Hash:                   hash,
		ScriptKind:             scriptKind,
	}
}

type ParseCache = RefCountCache[ParseCacheKey, *ast.SourceFile, FileHandle]

func NewParseCache(options RefCountCacheOptions) *ParseCache {
	return NewRefCountCache(
		options,
		func(key ParseCacheKey, fh FileHandle) *ast.SourceFile {
			if shouldUseSignatureExtractor(key.SourceFileParseOptions.FileName, key.ScriptKind, fh) {
				if sig := extractSignatures([]byte(fh.Content())); sig.ok {
					if file := parser.ParseSourceFile(key.SourceFileParseOptions, sig.declarations, key.ScriptKind); file != nil {
						file.Hash = fh.Hash()
						return file
					}
				}
			}

			file := parser.ParseSourceFile(key.SourceFileParseOptions, fh.Content(), key.ScriptKind)
			file.Hash = fh.Hash()
			return file
		},
		nil,
	)
}

func shouldUseSignatureExtractor(fileName string, scriptKind core.ScriptKind, fh FileHandle) bool {
	if fh.IsOverlay() || tspath.IsDeclarationFileName(fileName) {
		return false
	}

	switch scriptKind {
	case core.ScriptKindTS, core.ScriptKindTSX, core.ScriptKindJS, core.ScriptKindJSX:
		return true
	default:
		return false
	}
}
