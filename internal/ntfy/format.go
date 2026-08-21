package ntfy

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode"
)

const (
	MaxInput         = 10000
	MessageMaxLength = 1000
)

type ToolInfo struct {
	Title   string
	Message string
}

func FormatToolInfo(toolName string, toolInput map[string]interface{}) ToolInfo {
	var title, message string

	if toolName == "ExitPlanMode" {
		title = "Claude Code: Plan Review"
		if plan, ok := toolInput["plan"].(string); ok && strings.TrimSpace(plan) != "" {
			message = StripMarkdown(plan)
		} else {
			message = "(empty plan)"
		}
	} else {
		title = "Claude Code: " + toolName
		switch toolName {
		case "Bash":
			if cmd, ok := toolInput["command"].(string); ok {
				message = cmd
			} else {
				b, _ := json.Marshal(toolInput)
				message = string(b)
			}
		case "Read", "Write", "Edit":
			if fp, ok := toolInput["file_path"].(string); ok {
				message = fp
			} else {
				b, _ := json.Marshal(toolInput)
				message = string(b)
			}
		default:
			b, _ := json.Marshal(toolInput)
			message = string(b)
		}
	}

	if len(message) > MessageMaxLength {
		message = message[:MessageMaxLength] + "..."
	}

	return ToolInfo{Title: title, Message: message}
}

var (
	reFencedCode   = regexp.MustCompile("(?s)```[\\s\\S]*?(?:```|$)")
	reHorizRule    = regexp.MustCompile(`(?m)^[ \t]*(?:(?:-[ \t]*){3,}|(?:\*[ \t]*){3,}|(?:_[ \t]*){3,})$`)
	reHeaders      = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	reBlockQuotes  = regexp.MustCompile(`(?m)^(?:>[ \t]?)+`)
	reUnorderedList = regexp.MustCompile(`(?m)^[ \t]*[-*+] `)
	reOrderedList  = regexp.MustCompile(`(?m)^[ \t]*\d+\. `)
	reBlankLines   = regexp.MustCompile(`\n{2,}`)
)

func StripMarkdown(text string) string {
	if len(text) > MaxInput {
		text = text[:MaxInput]
	}

	result := reFencedCode.ReplaceAllString(text, "")
	result = reHorizRule.ReplaceAllString(result, "")
	result = reHeaders.ReplaceAllString(result, "")
	result = reBlockQuotes.ReplaceAllString(result, "")
	result = reUnorderedList.ReplaceAllString(result, "")
	result = reOrderedList.ReplaceAllString(result, "")

	result = stripInline(result)

	result = reBlankLines.ReplaceAllString(result, "\n")
	result = strings.TrimSpace(result)

	return result
}

func countRun(text string, pos int, ch byte) int {
	count := 0
	for pos+count < len(text) && text[pos+count] == ch {
		count++
	}
	return count
}

func isAlphanumeric(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

func isWhitespaceByte(ch byte) bool {
	return unicode.IsSpace(rune(ch))
}

func isAsciiPunctuation(ch byte) bool {
	return strings.ContainsRune(`!"#$%&'()*+,-./:;<=>?@[\]^_{|}~`, rune(ch))
}

func findBacktickCloser(text string, tickCount int, start int) int {
	i := start
	for i < len(text) {
		if text[i] == '`' {
			run := countRun(text, i, '`')
			if run == tickCount {
				return i
			}
			i += run
			continue
		}
		i++
	}
	return -1
}

func precomputePairs(str string, open byte, close byte) map[int]int {
	pairs := make(map[int]int)
	stack := []int{}
	i := 0
	for i < len(str) {
		if str[i] == '\\' && i+1 < len(str) && isAsciiPunctuation(str[i+1]) {
			i += 2
			continue
		}
		if str[i] == '`' {
			tickCount := countRun(str, i, '`')
			closeIdx := findBacktickCloser(str, tickCount, i+tickCount)
			if closeIdx != -1 {
				i = closeIdx + tickCount
				continue
			}
			i += tickCount
			continue
		}
		if str[i] == open {
			stack = append(stack, i)
		} else if str[i] == close && len(stack) > 0 {
			openIdx := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			pairs[openIdx] = i
		}
		i++
	}
	return pairs
}

func findStrikethroughCloser(text string, start int) int {
	i := start
	for i < len(text)-1 {
		if text[i] == '\\' && i+1 < len(text) && isAsciiPunctuation(text[i+1]) {
			i += 2
			continue
		}
		if text[i] == '~' && text[i+1] == '~' {
			return i
		}
		i++
	}
	return -1
}

func findEmphasisCloser(text string, ch byte, markerLen int, start int) int {
	i := start
	for i < len(text) {
		if text[i] == '\\' && i+1 < len(text) && isAsciiPunctuation(text[i+1]) {
			i += 2
			continue
		}
		if text[i] == ch {
			run := countRun(text, i, ch)
			if run >= markerLen && i > 0 && !isWhitespaceByte(text[i-1]) {
				return i
			}
			i += run
			continue
		}
		i++
	}
	return -1
}

type emphasisResult struct {
	output  string
	nextPos int
}

func handleEmphasis(text string, pos int) *emphasisResult {
	ch := text[pos]
	runLen := countRun(text, pos, ch)

	var prevChar byte
	if pos > 0 {
		prevChar = text[pos-1]
	}
	afterIdx := pos + runLen
	var afterChar byte
	if afterIdx < len(text) {
		afterChar = text[afterIdx]
	}

	isOpener := !isAlphanumeric(prevChar) && afterIdx < len(text) && !isWhitespaceByte(afterChar)
	if !isOpener {
		return nil
	}

	maxMarker := runLen
	if maxMarker > 3 {
		maxMarker = 3
	}

	for markerLen := maxMarker; markerLen >= 1; markerLen-- {
		searchFrom := pos + runLen
		for {
			idx := findEmphasisCloser(text, ch, markerLen, searchFrom)
			if idx == -1 {
				break
			}
			content := text[pos+markerLen : idx]
			if len(content) == 0 {
				searchFrom = idx + countRun(text, idx, ch)
				continue
			}
			return &emphasisResult{output: stripInline(content), nextPos: idx + markerLen}
		}
	}

	return nil
}

func stripInline(text string) string {
	bracketPairs := precomputePairs(text, '[', ']')
	parenPairs := precomputePairs(text, '(', ')')

	var out strings.Builder
	i := 0

	for i < len(text) {
		ch := text[i]

		// Escaped character
		if ch == '\\' && i+1 < len(text) && isAsciiPunctuation(text[i+1]) {
			out.WriteByte(text[i+1])
			i += 2
			continue
		}

		// Code spans
		if ch == '`' {
			tickCount := countRun(text, i, '`')
			closeIdx := findBacktickCloser(text, tickCount, i+tickCount)
			if closeIdx != -1 {
				out.WriteString(text[i+tickCount : closeIdx])
				i = closeIdx + tickCount
				continue
			}
			out.WriteString(text[i : i+tickCount])
			i += tickCount
			continue
		}

		// Images ![alt](url)
		if ch == '!' && i+1 < len(text) && text[i+1] == '[' {
			if closeBracket, ok := bracketPairs[i+1]; ok {
				if closeBracket+1 < len(text) && text[closeBracket+1] == '(' {
					if closeParen, ok := parenPairs[closeBracket+1]; ok {
						altText := text[i+2 : closeBracket]
						out.WriteString(stripInline(altText))
						i = closeParen + 1
						continue
					}
				}
			}
			out.WriteByte(ch)
			i++
			continue
		}

		// Links [text](url)
		if ch == '[' {
			if closeBracket, ok := bracketPairs[i]; ok {
				if closeBracket+1 < len(text) && text[closeBracket+1] == '(' {
					if closeParen, ok := parenPairs[closeBracket+1]; ok {
						linkText := text[i+1 : closeBracket]
						out.WriteString(stripInline(linkText))
						i = closeParen + 1
						continue
					}
				}
			}
			out.WriteByte(ch)
			i++
			continue
		}

		// Strikethrough ~~text~~
		if ch == '~' && i+1 < len(text) && text[i+1] == '~' {
			searchStart := i + 2
			closeIdx := findStrikethroughCloser(text, searchStart)
			if closeIdx != -1 {
				content := text[searchStart:closeIdx]
				if len(content) == 0 {
					out.WriteString("~~")
					i += 2
					continue
				}
				out.WriteString(stripInline(content))
				i = closeIdx + 2
				continue
			}
			out.WriteString("~~")
			i += 2
			continue
		}

		// Emphasis */** or _/__
		if ch == '*' || ch == '_' {
			result := handleEmphasis(text, i)
			if result != nil {
				out.WriteString(result.output)
				i = result.nextPos
				continue
			}
			runLen := countRun(text, i, ch)
			out.WriteString(text[i : i+runLen])
			i += runLen
			continue
		}

		out.WriteByte(ch)
		i++
	}

	return out.String()
}
