package signature

import (
	"bytes"
	"strings"
)

// Extract scans the source text and returns a minimal signature-only representation
// of the exported API surface. If no signatures can be extracted, it returns ok=false.
func Extract(src []byte) (string, bool) {
	out := processFile(src)
	if strings.TrimSpace(out) == "" {
		return "", false
	}
	return out, true
}

func isIdentChar(b byte) bool {
	if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_' || b == '$' {
		return true
	}
	return false
}

func skipWhitespaceAndComments(src []byte, i int) int {
	n := len(src)
	for i < n {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}
		if c == '/' && i+1 < n {
			nxt := src[i+1]
			if nxt == '/' {
				i += 2
				for i < n && src[i] != '\n' && src[i] != '\r' {
					i++
				}
				continue
			} else if nxt == '*' {
				i += 2
				for i < n-1 {
					if src[i] == '*' && src[i+1] == '/' {
						i += 2
						break
					}
					i++
				}
				continue
			}
		}
		break
	}
	return i
}

func skipString(src []byte, i int) int {
	n := len(src)
	quote := src[i]
	i++
	for i < n {
		c := src[i]
		if c == '\\' {
			i += 2
			continue
		}
		if c == quote {
			i++
			break
		}
		i++
	}
	return i
}

func skipRegex(src []byte, i int) int {
	n := len(src)
	i++
	inClass := false
	for i < n {
		c := src[i]
		if c == '\\' {
			i += 2
			continue
		}
		if c == '[' {
			if !inClass {
				inClass = true
			}
			i++
			continue
		}
		if c == ']' && inClass {
			inClass = false
			i++
			continue
		}
		if c == '/' && !inClass {
			i++
			break
		}
		i++
	}
	for i < n {
		c := src[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			i++
		} else {
			break
		}
	}
	return i
}

func skipTemplate(src []byte, i int) int {
	n := len(src)
	i++
	for i < n {
		c := src[i]
		if c == '\\' {
			i += 2
			continue
		}
		if c == '`' {
			i++
			break
		}
		if c == '$' && i+1 < n && src[i+1] == '{' {
			i = skipBraces(src, i+1)
			continue
		}
		i++
	}
	return i
}

func skipBraces(src []byte, i int) int {
	n := len(src)
	depth := 1
	i++
	for i < n && depth > 0 {
		c := src[i]
		if c == '/' && i+1 < n {
			nxt := src[i+1]
			if nxt == '/' {
				i += 2
				for i < n && src[i] != '\n' && src[i] != '\r' {
					i++
				}
				continue
			} else if nxt == '*' {
				i += 2
				for i < n-1 {
					if src[i] == '*' && src[i+1] == '/' {
						i += 2
						break
					}
					i++
				}
				continue
			}
		}
		if c == '"' || c == '\'' {
			i = skipString(src, i)
			continue
		}
		if c == '`' {
			i = skipTemplate(src, i)
			continue
		}
		if c == '/' {
			i = skipRegex(src, i)
			continue
		}
		if c == '{' {
			depth++
			i++
			continue
		}
		if c == '}' {
			depth--
			i++
			continue
		}
		i++
	}
	return i
}

func scanAngleBrackets(src []byte, i int) int {
	n := len(src)
	depth := 0
	for i < n {
		c := src[i]
		if c == '<' {
			depth++
			i++
			continue
		}
		if c == '>' {
			depth--
			i++
			if depth == 0 {
				break
			}
			continue
		}
		if c == '"' || c == '\'' {
			i = skipString(src, i)
			continue
		}
		if c == '`' {
			i = skipTemplate(src, i)
			continue
		}
		if c == '/' && i+1 < n {
			nxt := src[i+1]
			if nxt == '/' {
				i += 2
				for i < n && src[i] != '\n' && src[i] != '\r' {
					i++
				}
				continue
			} else if nxt == '*' {
				i += 2
				for i < n-1 {
					if src[i] == '*' && src[i+1] == '/' {
						i += 2
						break
					}
					i++
				}
				continue
			}
			i = skipRegex(src, i)
			continue
		}
		i++
	}
	return i
}

func scanInterfaceOrEnum(src []byte, kwStart int, kw string) int {
	n := len(src)
	i := kwStart + len(kw)
	i = skipWhitespaceAndComments(src, i)
	for i < n {
		c := src[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '$' {
			i++
			continue
		}
		break
	}
	i = skipWhitespaceAndComments(src, i)
	if i < n && src[i] == '<' {
		i = scanAngleBrackets(src, i)
		i = skipWhitespaceAndComments(src, i)
	}
	for i < n {
		c := src[i]
		if c == '{' {
			return skipBraces(src, i)
		}
		if c == ';' {
			return i + 1
		}
		if c == '"' || c == '\'' {
			i = skipString(src, i)
			continue
		}
		if c == '`' {
			i = skipTemplate(src, i)
			continue
		}
		if c == '/' && i+1 < n {
			nxt := src[i+1]
			if nxt == '/' {
				i += 2
				for i < n && src[i] != '\n' && src[i] != '\r' {
					i++
				}
				continue
			} else if nxt == '*' {
				i += 2
				for i < n-1 {
					if src[i] == '*' && src[i+1] == '/' {
						i += 2
						break
					}
					i++
				}
				continue
			}
			i = skipRegex(src, i)
			continue
		}
		i++
	}
	return i
}

func scanTypeAlias(src []byte, kwStart int) int {
	n := len(src)
	i := kwStart + len("type")
	i = skipWhitespaceAndComments(src, i)
	for i < n {
		c := src[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '$' {
			i++
			continue
		}
		break
	}
	i = skipWhitespaceAndComments(src, i)
	if i < n && src[i] == '<' {
		i = scanAngleBrackets(src, i)
		i = skipWhitespaceAndComments(src, i)
	}
	if i < n && src[i] == '=' {
		i++
	}
	depthParen := 0
	depthBrace := 0
	depthBracket := 0
	for i < n {
		c := src[i]
		if c == '"' || c == '\'' {
			i = skipString(src, i)
			continue
		}
		if c == '`' {
			i = skipTemplate(src, i)
			continue
		}
		if c == '/' && i+1 < n {
			nxt := src[i+1]
			if nxt == '/' {
				i += 2
				for i < n && src[i] != '\n' && src[i] != '\r' {
					i++
				}
				continue
			} else if nxt == '*' {
				i += 2
				for i < n-1 {
					if src[i] == '*' && src[i+1] == '/' {
						i += 2
						break
					}
					i++
				}
				continue
			}
			i = skipRegex(src, i)
			continue
		}
		switch c {
		case '(':
			depthParen++
		case ')':
			if depthParen > 0 {
				depthParen--
			}
		case '{':
			depthBrace++
		case '}':
			if depthBrace > 0 {
				depthBrace--
			}
		case '[':
			depthBracket++
		case ']':
			if depthBracket > 0 {
				depthBracket--
			}
		case ';':
			if depthParen == 0 && depthBrace == 0 && depthBracket == 0 {
				return i + 1
			}
		}
		i++
	}
	return i
}

func scanFunctionOrClass(src []byte, kwStart int, kw string) (int, int, bool) {
	n := len(src)
	startIdx := kwStart
	i := kwStart + len(kw)
	i = skipWhitespaceAndComments(src, i)
	for i < n {
		c := src[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '$' {
			i++
			continue
		}
		break
	}
	i = skipWhitespaceAndComments(src, i)
	if i < n && src[i] == '<' {
		i = scanAngleBrackets(src, i)
		i = skipWhitespaceAndComments(src, i)
	}
	parenDepth := 0
	angleDepth := 0
	hasBody := false
	for i < n {
		c := src[i]
		if c == '"' || c == '\'' {
			i = skipString(src, i)
			continue
		}
		if c == '`' {
			i = skipTemplate(src, i)
			continue
		}
		if c == '/' && i+1 < n {
			nxt := src[i+1]
			if nxt == '/' {
				i += 2
				for i < n && src[i] != '\n' && src[i] != '\r' {
					i++
				}
				continue
			} else if nxt == '*' {
				i += 2
				for i < n-1 {
					if src[i] == '*' && src[i+1] == '/' {
						i += 2
						break
					}
					i++
				}
				continue
			}
			i = skipRegex(src, i)
			continue
		}
		if c == '(' {
			parenDepth++
			i++
			continue
		}
		if c == ')' {
			if parenDepth > 0 {
				parenDepth--
			}
			i++
			continue
		}
		if c == '<' {
			angleDepth++
			i++
			continue
		}
		if c == '>' {
			if angleDepth > 0 {
				angleDepth--
			}
			i++
			continue
		}
		if c == '{' && parenDepth == 0 && angleDepth == 0 {
			hasBody = true
			return startIdx, i, hasBody
		}
		if c == ';' && parenDepth == 0 && angleDepth == 0 {
			return startIdx, i + 1, false
		}
		i++
	}
	return startIdx, n, false
}

func processFile(src []byte) string {
	var outParts []string
	i := 0
	n := len(src)
	for i < n {
		idx := bytes.Index(src[i:], []byte("export"))
		if idx < 0 {
			break
		}
		pos := i + idx
		if pos > 0 {
			b := src[pos-1]
			if isIdentChar(b) {
				i = pos + 1
				continue
			}
		}
		if pos+len("export") < n {
			b := src[pos+len("export")]
			if isIdentChar(b) {
				i = pos + 1
				continue
			}
		}
		j := pos + len("export")
		j = skipWhitespaceAndComments(src, j)
		defaultFlag := false
		if j+len("default") <= n && bytes.HasPrefix(src[j:], []byte("default")) {
			before := byte(' ')
			after := byte(' ')
			if j > 0 {
				before = src[j-1]
			}
			if j+len("default") < n {
				after = src[j+len("default")]
			}
			if !isIdentChar(before) && !isIdentChar(after) {
				defaultFlag = true
				j += len("default")
				j = skipWhitespaceAndComments(src, j)
			}
		}
		kwStart := j
		kwEnd := kwStart
		for kwEnd < n {
			c := src[kwEnd]
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '$' {
				kwEnd++
				continue
			}
			break
		}
		kw := string(src[kwStart:kwEnd])
		switch kw {
		case "interface", "enum":
			end := scanInterfaceOrEnum(src, kwStart, kw)
			segment := string(src[pos:end])
			outParts = append(outParts, segment)
			i = end
			continue
		case "type":
			end := scanTypeAlias(src, kwStart)
			segment := string(src[pos:end])
			outParts = append(outParts, segment)
			i = end
			continue
		case "function", "class":
			startIdx, sigEnd, hasBody := scanFunctionOrClass(src, kwStart, kw)
			prefix := "export "
			if !defaultFlag {
				prefix += "declare "
			} else {
				prefix += "default "
			}
			signature := string(bytes.TrimSpace(src[startIdx:sigEnd]))
			if !strings.HasSuffix(signature, ";") {
				signature += ";"
			}
			outParts = append(outParts, prefix+signature)
			if hasBody {
				bodyStart := sigEnd
				if bodyStart < n && src[bodyStart] == '{' {
					bodyEnd := skipBraces(src, bodyStart)
					i = bodyEnd
				} else {
					i = sigEnd
				}
			} else {
				i = sigEnd
			}
			continue
		default:
			j := kwStart
			for j < n {
				c := src[j]
				if c == ';' {
					j++
					break
				}
				if c == '{' {
					j = skipBraces(src, j)
					continue
				}
				if c == '"' || c == '\'' {
					j = skipString(src, j)
					continue
				}
				if c == '`' {
					j = skipTemplate(src, j)
					continue
				}
				if c == '/' && j+1 < n {
					nxt := src[j+1]
					if nxt == '/' {
						j += 2
						for j < n && src[j] != '\n' && src[j] != '\r' {
							j++
						}
						continue
					} else if nxt == '*' {
						j += 2
						for j < n-1 {
							if src[j] == '*' && src[j+1] == '/' {
								j += 2
								break
							}
							j++
						}
						continue
					}
					j = skipRegex(src, j)
					continue
				}
				j++
			}
			i = j
			continue
		}
	}
	if len(outParts) == 0 {
		return ""
	}
	return strings.Join(outParts, "\n") + "\n"
}
