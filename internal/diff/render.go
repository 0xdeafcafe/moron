package diff

import (
	"fmt"
	"path/filepath"
	"strings"

	chroma "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderStyles holds the styles for rendering diffs.
type RenderStyles struct {
	Added      lipgloss.Style
	Removed    lipgloss.Style
	Context    lipgloss.Style
	HunkHeader lipgloss.Style
	LineNum    lipgloss.Style
	Selected   lipgloss.Style
}

// DefaultRenderStyles returns the default diff rendering styles.
func DefaultRenderStyles() RenderStyles {
	return RenderStyles{
		Added:      lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
		Removed:    lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		Context:    lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
		HunkHeader: lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
		LineNum:    lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		Selected:   lipgloss.NewStyle().Background(lipgloss.Color("8")),
	}
}

// RenderedLine holds a rendered diff line with metadata.
type RenderedLine struct {
	Text       string
	IsHunkHead bool
	HunkIdx    int
	LineIdx    int // index within hunk, -1 for hunk headers
}

// RenderFileDiff renders a complete file diff into styled lines.
func RenderFileDiff(fd FileDiff, styles RenderStyles) []RenderedLine {
	return renderFileDiff(fd, styles, "")
}

// RenderFileDiffHighlighted renders a file diff with syntax highlighting.
func RenderFileDiffHighlighted(fd FileDiff, styles RenderStyles, fileName string) []RenderedLine {
	return renderFileDiff(fd, styles, fileName)
}

func renderFileDiff(fd FileDiff, styles RenderStyles, fileName string) []RenderedLine {
	var rendered []RenderedLine

	if fd.IsBinary {
		rendered = append(rendered, RenderedLine{
			Text:    styles.HunkHeader.Render("Binary file differs"),
			LineIdx: -1,
		})
		return rendered
	}

	// Set up syntax highlighter
	var highlighter *syntaxHighlighter
	if fileName == "" {
		fileName = fd.NewName
	}
	if fileName != "" {
		highlighter = newSyntaxHighlighter(fileName)
	}

	for hi, hunk := range fd.Hunks {
		rendered = append(rendered, RenderedLine{
			Text:       styles.HunkHeader.Render(hunk.Header),
			IsHunkHead: true,
			HunkIdx:    hi,
			LineIdx:    -1,
		})

		for li, line := range hunk.Lines {
			oldStr := "   "
			newStr := "   "

			if line.OldNum > 0 {
				oldStr = fmt.Sprintf("%3d", line.OldNum)
			}
			if line.NewNum > 0 {
				newStr = fmt.Sprintf("%3d", line.NewNum)
			}

			lineNums := styles.LineNum.Render(oldStr + " " + newStr + " ")

			var text string
			switch line.Type {
			case LineAdded:
				content := highlightContent(highlighter, line.Content, styles.Added)
				text = lineNums + styles.Added.Render("+") + content
			case LineRemoved:
				content := highlightContent(highlighter, line.Content, styles.Removed)
				text = lineNums + styles.Removed.Render("-") + content
			case LineContext:
				content := highlightContent(highlighter, line.Content, styles.Context)
				text = lineNums + styles.Context.Render(" ") + content
			}

			rendered = append(rendered, RenderedLine{
				Text:    text,
				HunkIdx: hi,
				LineIdx: li,
			})
		}
	}

	return rendered
}

type syntaxHighlighter struct {
	lexer chroma.Lexer
	style *chroma.Style
}

func newSyntaxHighlighter(fileName string) *syntaxHighlighter {
	// Use just the base name for lexer matching
	base := filepath.Base(fileName)
	lexer := lexers.Match(base)
	if lexer == nil {
		return nil
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	return &syntaxHighlighter{
		lexer: lexer,
		style: style,
	}
}

func (h *syntaxHighlighter) tokenize(content string) []chroma.Token {
	tokens, err := chroma.Tokenise(h.lexer, nil, content)
	if err != nil {
		return nil
	}
	return tokens
}

func highlightContent(h *syntaxHighlighter, content string, fallbackStyle lipgloss.Style) string {
	if h == nil {
		return fallbackStyle.Render(content)
	}

	tokens := h.tokenize(content)
	if len(tokens) == 0 {
		return fallbackStyle.Render(content)
	}

	var b strings.Builder
	for _, tok := range tokens {
		val := strings.TrimRight(tok.Value, "\n\r")
		if val == "" {
			continue
		}

		entry := h.style.Get(tok.Type)
		if entry.Colour.IsSet() {
			hex := entry.Colour.String()
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
			if entry.Bold == chroma.Yes {
				style = style.Bold(true)
			}
			if entry.Italic == chroma.Yes {
				style = style.Italic(true)
			}
			b.WriteString(style.Render(val))
		} else {
			b.WriteString(fallbackStyle.Render(val))
		}
	}

	return b.String()
}

// RenderFileDiffPlain returns an unstyled string representation.
func RenderFileDiffPlain(fd FileDiff) string {
	var b strings.Builder
	for _, hunk := range fd.Hunks {
		b.WriteString(hunk.Header)
		b.WriteString("\n")
		for _, line := range hunk.Lines {
			switch line.Type {
			case LineAdded:
				b.WriteString("+" + line.Content + "\n")
			case LineRemoved:
				b.WriteString("-" + line.Content + "\n")
			case LineContext:
				b.WriteString(" " + line.Content + "\n")
			}
		}
	}
	return b.String()
}
