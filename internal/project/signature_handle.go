package project

import (
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/zeebo/xxh3"
)

type signatureFileHandle struct {
	fileBase
	kind core.ScriptKind
}

func newSignatureFileHandle(fileName string, content string, kind core.ScriptKind) *signatureFileHandle {
	return &signatureFileHandle{
		fileBase: fileBase{
			fileName: fileName,
			content:  content,
			hash:     xxh3.HashString128(content),
		},
		kind: kind,
	}
}

func (s *signatureFileHandle) Version() int32        { return 0 }
func (s *signatureFileHandle) MatchesDiskText() bool { return false }
func (s *signatureFileHandle) IsOverlay() bool       { return false }
func (s *signatureFileHandle) Kind() core.ScriptKind { return s.kind }
