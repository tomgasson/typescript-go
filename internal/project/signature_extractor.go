package project

import (
	"bytes"
	"strings"
)

func extractModuleSignature(src []byte) (string, bool) {
	out := processSignatureFile(src)
	return out, out != ""
}

// The functions below provide a lightweight signature extractor for TypeScript/JavaScript files.
// They are adapted from a standalone prototype and intentionally avoid a full parse so they can
// be used for non-focused files where only exported declarations are required.

// isIdentChar reports whether b is a valid identifier character for simple scanning.
// We treat ASCII letters, digits, underscore and dollar as identifier characters.
func isIdentChar(b byte) bool {
	if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_' || b == '$' {
		return true
	}
	return false
}

// skipWhitespaceAndComments advances i past whitespace and both line and block comments.
func skipWhitespaceAndComments(src []byte, i int) int {
	n := len(src)
	for i < n {
		c := src[i]
		// Skip whitespace
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}
		// Skip line or block comments
		if c == '/' && i+1 < n {
			nxt := src[i+1]
			if nxt == '/' {
				// Line comment
				i += 2
				for i < n && src[i] != '\n' && src[i] != '\r' {
					i++
				}
				continue
			} else if nxt == '*' {
				// Block comment
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

// skipString skips over a single or double quoted string literal starting at src[i].
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

// skipRegex skips over a JavaScript/TypeScript regular expression literal starting at src[i].
func skipRegex(src []byte, i int) int {
	n := len(src)
	i++ // skip opening /
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
	// Skip trailing flags (alphabetic)
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

// skipTemplate skips over a template literal starting at src[i].
func skipTemplate(src []byte, i int) int {
	n := len(src)
	// assume src[i] == '`'
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
			// enter expression
			i = skipBraces(src, i+1)
			continue
		}
		i++
	}
	return i
}

// skipBraces skips over a balanced set of curly braces starting at src[i] == '{'.
func skipBraces(src []byte, i int) int {
	n := len(src)
	// assume src[i] == '{'
	depth := 1
	i++
	for i < n && depth > 0 {
		c := src[i]
		// Handle comments
		if c == '/' && i+1 < n {
			nxt := src[i+1]
			if nxt == '/' {
				// Line comment
				i += 2
				for i < n && src[i] != '\n' && src[i] != '\r' {
					i++
				}
				continue
			} else if nxt == '*' {
				// Block comment
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
		// Handle string literals
		if c == '"' || c == '\'' {
			i = skipString(src, i)
			continue
		}
		// Handle template literals
		if c == '`' {
			i = skipTemplate(src, i)
			continue
		}
		// Handle regex literals (heuristic: treat '/' as regex)
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

// scanAngleBrackets skips over a balanced sequence of angle brackets starting at src[i] == '<'.
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
				// line comment
				i += 2
				for i < n && src[i] != '\n' && src[i] != '\r' {
					i++
				}
				continue
			} else if nxt == '*' {
				// block comment
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
			// else treat as regex start
			i = skipRegex(src, i)
			continue
		}
		i++
	}
	return i
}

// scanInterfaceOrEnum scans an interface or enum declaration starting at kwStart.
// kw should be either "interface" or "enum". It returns the index just past the
// end of the declaration, including the closing brace or semicolon.
func scanInterfaceOrEnum(src []byte, kwStart int, kw string) int {
	n := len(src)
	i := kwStart + len(kw)
	// Skip whitespace/comments
	i = skipWhitespaceAndComments(src, i)
	// Skip identifier
	for i < n {
		c := src[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '$' {
			i++
			continue
		}
		break
	}
	// Skip whitespace/comments
	i = skipWhitespaceAndComments(src, i)
	// Skip generic parameters
	if i < n && src[i] == '<' {
		i = scanAngleBrackets(src, i)
		i = skipWhitespaceAndComments(src, i)
	}
	// Skip until opening brace or semicolon
	for i < n {
		c := src[i]
		if c == '{' {
			// capture body
			return skipBraces(src, i)
		}
		if c == ';' {
			return i + 1
		}
		// Skip strings, templates and comments
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
			// regex literal
			i = skipRegex(src, i)
			continue
		}
		i++
	}
	return i
}

// scanTypeAlias scans a type alias declaration starting at kwStart (pointing to 'type').
// It returns the index just after the terminating semicolon.
func scanTypeAlias(src []byte, kwStart int) int {
	n := len(src)
	i := kwStart + len("type")
	// Skip whitespace/comments
	i = skipWhitespaceAndComments(src, i)
	// Skip identifier
	for i < n {
		c := src[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '$' {
			i++
			continue
		}
		break
	}
	// Skip whitespace/comments
	i = skipWhitespaceAndComments(src, i)
	// Skip generic parameters
	if i < n && src[i] == '<' {
		i = scanAngleBrackets(src, i)
		i = skipWhitespaceAndComments(src, i)
	}
	// Skip assignment operator '=' if present
	if i < n && src[i] == '=' {
		i++
	}
	// Scan until semicolon at top-level depth (no unclosed parens/braces/brackets)
	depthParen := 0
	depthBrace := 0
	depthBracket := 0
	for i < n {
		c := src[i]
		// Skip strings, templates, comments
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
		// Track depths
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

// scanFunctionOrClass scans a function or class declaration starting at kwStart.
// kw should be either "function" or "class". It returns (startIdx, signatureEndIdx, hasBody).
// startIdx points to the start of the keyword (kwStart), signatureEndIdx points to the
// character immediately before the opening brace if there is a body, or just after
// the terminating semicolon if there is no body. hasBody is true if a body is present.
func scanFunctionOrClass(src []byte, kwStart int, kw string) (int, int, bool) {
	n := len(src)
	startIdx := kwStart
	i := kwStart + len(kw)
	// Skip whitespace/comments
	i = skipWhitespaceAndComments(src, i)
	// Skip optional identifier
	for i < n {
		c := src[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '$' {
			i++
			continue
		}
		break
	}
	// Skip whitespace/comments
	i = skipWhitespaceAndComments(src, i)
	// Skip generic parameters
	if i < n && src[i] == '<' {
		i = scanAngleBrackets(src, i)
		i = skipWhitespaceAndComments(src, i)
	}
	// Track parentheses and angle brackets in parameter lists or heritage clauses
	parenDepth := 0
	angleDepth := 0
	hasBody := false
	for i < n {
		c := src[i]
		// Skip strings, templates, comments
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
		// Detect body start
		if c == '{' && parenDepth == 0 && angleDepth == 0 {
			hasBody = true
			return startIdx, i, hasBody
		}
		// Detect end of signature with semicolon
		if c == ';' && parenDepth == 0 && angleDepth == 0 {
			return startIdx, i + 1, false
		}
		i++
	}
	return startIdx, n, false
}

// processSignatureFile processes the source and returns the generated declarations string.
func processSignatureFile(src []byte) string {
	var outParts []string
	i := 0
	n := len(src)
	for i < n {
		idx := bytes.Index(src[i:], []byte("export"))
		if idx < 0 {
			break
		}
		pos := i + idx
		// Ensure keyword boundaries
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
		// Move past 'export'
		j := pos + len("export")
		// Skip whitespace/comments
		j = skipWhitespaceAndComments(src, j)
		// Check for 'default'
		defaultFlag := false
		if j+len("default") <= n && bytes.HasPrefix(src[j:], []byte("default")) {
			// ensure boundaries
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
		// Extract keyword after export
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
		// Dispatch based on keyword
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
			// Build prefix
			prefix := "export "
			if !defaultFlag {
				prefix += "declare "
			} else {
				prefix += "default "
			}
			signature := strings.TrimSpace(string(src[startIdx:sigEnd]))
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
			// Skip unsupported exports (const/let/var/namespace/etc.)
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
				// Skip strings, templates, comments
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
