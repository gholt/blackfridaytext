// This renderer is available at http://github.com/gholt/blackfridaytext and is
// Copyright © 2014 Gregory Holt <greg@brim.net>.
//
// Distributed under the Simplified BSD License.
//
// See README.md for details.

// Package blackfridaytext contains an experimental text renderer for the
// Blackfriday Markdown Processor http://github.com/russross/blackfriday.
package blackfridaytext

import (
	"bytes"
	"code.google.com/p/go.crypto/ssh/terminal"
	"fmt"
	"github.com/russross/blackfriday"
	"os"
	"strings"
)

const (
	_BLACKFRIDAY_EXTENSIONS = blackfriday.EXTENSION_NO_INTRA_EMPHASIS | blackfriday.EXTENSION_TABLES | blackfriday.EXTENSION_FENCED_CODE | blackfriday.EXTENSION_AUTOLINK | blackfriday.EXTENSION_STRIKETHROUGH
)

const (
	_                         byte = iota // skip 0 NUL
	_LINE_BREAK_MARKER                    // 1 SOH
	_NBSP_MARKER                          // 2 STX
	_INDENT_START_MARKER                  // 3 ETX
	_INDENT_FIRST_MARKER                  // 4 EOT
	_INDENT_SUBSEQUENT_MARKER             // 5 ENQ
	_INDENT_STOP_MARKER                   // 6 ACK
	_TABLE_ROW_MARKER                     // 7 BEL
	_TABLE_CELL_MARKER                    // 8 BS
	_                                     // 9 TAB
	_                                     // 10 LF
	_HRULE_MARKER                         // 11 VT
)

type ansiEscapeCodes struct {
	Reset                                                 []byte
	Bold                                                  []byte
	Black, Red, Green, Yellow, Blue, Magenta, Cyan, White []byte
}

var ansiEscape = ansiEscapeCodes{
	Reset:   []byte{27, '[', '0', 'm'},
	Bold:    []byte{27, '[', '1', 'm'},
	Black:   []byte{27, '[', '3', '0', 'm'},
	Red:     []byte{27, '[', '3', '1', 'm'},
	Green:   []byte{27, '[', '3', '2', 'm'},
	Yellow:  []byte{27, '[', '3', '3', 'm'},
	Blue:    []byte{27, '[', '3', '4', 'm'},
	Magenta: []byte{27, '[', '3', '5', 'm'},
	Cyan:    []byte{27, '[', '3', '6', 'm'},
	White:   []byte{27, '[', '3', '7', 'm'},
}

func GetWidth() int {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0600)
	if err != nil {
		tty = os.Stdout
	}
	width, _, err := terminal.GetSize(int(tty.Fd()))
	if err != nil {
		return 79
	}
	return width - 1
}

// MarkdownToText parses the markdown text using the Blackfriday Markdown Processor and an internal renderer to return any metadata and the formatted text.
//
// The width int may be a positive integer for a specific width, 0 for the default width (attempted to get from terminal, 79 otherwise), or a negative number for a width relative to the default.
//
// The color bool is to indicate whether it is okay to emit ANSI color escape sequences or not.
//
// The metadata is a [][]string where each []string will have two elements, the metadata item name and the value. Metadata is an extension of standard Markdown and is documented at https://github.com/fletcher/MultiMarkdown/wiki/MultiMarkdown-Syntax-Guide#metadata -- this implementation currently differs in that it requires the trailing two spaces on each metadata line and doesn't support multiline values.
func MarkdownToText(markdown []byte, width int, color bool) ([][]string, []byte) {
	metadata := make([][]string, 0)
	position := 0
	for _, line := range bytes.Split(markdown, []byte("\n")) {
		if bytes.HasSuffix(line, []byte("  ")) {
			colon := bytes.Index(line, []byte(":"))
			if colon != -1 {
				metadata = append(metadata, []string{
					strings.Trim(string(line[:colon]), " "),
					strings.Trim(string(line[colon+1:]), " ")})
				position += len(line) + 1
			}
		} else {
			break
		}
	}
	text := markdown[position:]
	if len(text) == 0 {
		return metadata, []byte{}
	}
	return metadata, MarkdownToTextNoMetadata(text, width, color)
}

// MarkdownToTextNoMetadata is the same as MarkdownToText only skipping the detection and parsing of any leading metadata.
func MarkdownToTextNoMetadata(markdown []byte, width int, color bool) []byte {
	if width < 1 {
		width = GetWidth() + width
	}
	r := &renderer{width: width, color: color}
	text := blackfriday.Markdown(markdown, r, _BLACKFRIDAY_EXTENSIONS)
	for r.headerLevel > 0 {
		text = append(text, _INDENT_STOP_MARKER)
		r.headerLevel--
	}
	if len(text) > 0 {
		text = bytes.Replace(text, []byte(" \n"), []byte(" "), -1)
		text = bytes.Replace(text, []byte("\n"), []byte(" "), -1)
		text = reflow(text, []byte{}, []byte{}, r.width)
		text = bytes.Replace(text, []byte{_NBSP_MARKER}, []byte(" "), -1)
		text = bytes.Replace(
			text, []byte{_LINE_BREAK_MARKER}, []byte("\n"), -1)
	}
	return text
}

type renderer struct {
	width       int
	color       bool
	headerLevel int
}

func (r *renderer) BlockCode(out *bytes.Buffer, text []byte, lang string) {
	textLength := len(text)
	if textLength > 0 && text[textLength-1] == '\n' {
		text = text[:textLength-1]
	}
	r.ensureBlankLine(out)
	for _, line := range bytes.Split(text, []byte("\n")) {
		if r.color {
			out.Write(ansiEscape.Green)
		}
		out.Write(bytes.Replace(line, []byte(" "), []byte{_NBSP_MARKER}, -1))
		if r.color {
			out.Write(ansiEscape.Reset)
		}
		out.WriteByte(_LINE_BREAK_MARKER)
	}
	r.ensureBlankLine(out)
}

func (r *renderer) BlockQuote(out *bytes.Buffer, text []byte) {
	r.ensureBlankLine(out)
	out.WriteByte(_INDENT_START_MARKER)
	out.WriteString("> ")
	out.WriteByte(_INDENT_FIRST_MARKER)
	out.WriteString("> ")
	out.WriteByte(_INDENT_SUBSEQUENT_MARKER)
	out.Write(bytes.Trim(text, string([]byte{_LINE_BREAK_MARKER})))
	out.WriteByte(_INDENT_STOP_MARKER)
}

func (r *renderer) BlockHtml(out *bytes.Buffer, text []byte) {
	r.ensureBlankLine(out)
	out.Write(bytes.Replace(
		text, []byte("\n"), []byte{_LINE_BREAK_MARKER}, -1))
}

func (r *renderer) Header(
	out *bytes.Buffer, text func() bool, level int, id string) {
	marker := out.Len()
	r.ensureBlankLine(out)
	lastHeaderLevel := r.headerLevel
	level--
	for r.headerLevel > level {
		out.WriteByte(_INDENT_STOP_MARKER)
		r.headerLevel--
	}
	out.WriteByte(_INDENT_START_MARKER)
	out.WriteString("--[ ")
	out.WriteByte(_INDENT_FIRST_MARKER)
	out.WriteString("    ")
	out.WriteByte(_INDENT_SUBSEQUENT_MARKER)
	if r.color {
		out.Write(ansiEscape.Bold)
	}
	if !text() {
		out.Truncate(marker)
		r.headerLevel = lastHeaderLevel
		return
	}
	if r.color {
		out.Write(ansiEscape.Reset)
	}
	out.WriteByte(_NBSP_MARKER)
	out.WriteString("]--")
	out.WriteByte(_INDENT_STOP_MARKER)
	for r.headerLevel <= level {
		out.WriteByte(_INDENT_START_MARKER)
		out.WriteString("    ")
		out.WriteByte(_INDENT_FIRST_MARKER)
		out.WriteString("    ")
		out.WriteByte(_INDENT_SUBSEQUENT_MARKER)
		r.headerLevel++
	}
	r.ensureBlankLine(out)
}

func (r *renderer) HRule(out *bytes.Buffer) {
	r.ensureBlankLine(out)
	out.WriteByte(_HRULE_MARKER)
	out.WriteByte('-')
	r.ensureBlankLine(out)
}

func (r *renderer) List(out *bytes.Buffer, text func() bool, flags int) {
	marker := out.Len()
	r.ensureNewLine(out)
	if !text() {
		out.Truncate(marker)
		return
	}
}

func (r *renderer) ListItem(out *bytes.Buffer, text []byte, flags int) {
	r.ensureNewLine(out)
	out.WriteByte(_INDENT_START_MARKER)
	out.WriteString("  * ")
	out.WriteByte(_INDENT_FIRST_MARKER)
	out.WriteString("    ")
	out.WriteByte(_INDENT_SUBSEQUENT_MARKER)
	out.Write(bytes.Trim(text, string([]byte{_LINE_BREAK_MARKER})))
	out.WriteByte(_INDENT_STOP_MARKER)
}

func (r *renderer) Paragraph(out *bytes.Buffer, text func() bool) {
	r.ensureBlankLine(out)
	marker := out.Len()
	if !text() {
		out.Truncate(marker)
	}
}

func (r *renderer) Table(
	out *bytes.Buffer, header []byte, body []byte, columnData []int) {
	r.ensureBlankLine(out)
	headerRows := make([][][]byte, 0)
	for _, row := range bytes.Split(
		header[:len(header)-1], []byte{_TABLE_ROW_MARKER}) {
		headerRow := make([][]byte, 0)
		for _, cell := range bytes.Split(
			row[:len(row)-1], []byte{_TABLE_CELL_MARKER}) {
			headerRow = append(headerRow, cell)
		}
		headerRows = append(headerRows, headerRow)
	}
	bodyRows := make([][][]byte, 0)
	for _, row := range bytes.Split(
		body[:len(body)-1], []byte{_TABLE_ROW_MARKER}) {
		if len(row) == 0 {
			continue
		}
		bodyRow := make([][]byte, 0)
		for _, cell := range bytes.Split(
			row[:len(row)-1], []byte{_TABLE_CELL_MARKER}) {
			bodyRow = append(bodyRow, cell)
		}
		bodyRows = append(bodyRows, bodyRow)
	}
	widths := make([]int, len(headerRows[0]))
	for _, row := range headerRows {
		for column, cell := range row {
			columnWidth := len(string(cell))
			if columnWidth > widths[column] {
				widths[column] = columnWidth
			}
		}
	}
	for _, row := range bodyRows {
		for column, cell := range row {
			columnWidth := len(string(cell))
			if columnWidth > widths[column] {
				widths[column] = columnWidth
			}
		}
	}
	for _, width := range widths {
		out.WriteByte('+')
		for ; width > -2; width-- {
			out.WriteByte('-')
		}
	}
	out.WriteByte('+')
	out.WriteByte(_LINE_BREAK_MARKER)
	for _, row := range headerRows {
		for column, cell := range row {
			out.WriteByte('|')
			out.WriteByte(' ')
			out.Write(cell)
			for i := len(string(cell)); i < widths[column]; i++ {
				out.WriteByte(' ')
			}
			out.WriteByte(' ')
		}
		out.WriteByte('|')
		out.WriteByte(_LINE_BREAK_MARKER)
	}
	for _, width := range widths {
		out.WriteByte('+')
		for ; width > -2; width-- {
			out.WriteByte('-')
		}
	}
	out.WriteByte('+')
	out.WriteByte(_LINE_BREAK_MARKER)
	for _, row := range bodyRows {
		for column, cell := range row {
			out.WriteByte('|')
			out.WriteByte(' ')
			out.Write(cell)
			for i := len(string(cell)); i < widths[column]; i++ {
				out.WriteByte(' ')
			}
			out.WriteByte(' ')
		}
		out.WriteByte('|')
		out.WriteByte(_LINE_BREAK_MARKER)
	}
	for _, width := range widths {
		out.WriteByte('+')
		for ; width > -2; width-- {
			out.WriteByte('-')
		}
	}
	out.WriteByte('+')
	out.WriteByte(_LINE_BREAK_MARKER)
}

func (r *renderer) TableRow(out *bytes.Buffer, text []byte) {
	out.Write(text)
	out.WriteByte(_TABLE_ROW_MARKER)
}

func (r *renderer) TableHeaderCell(out *bytes.Buffer, text []byte, flags int) {
	out.Write(text)
	out.WriteByte(_TABLE_CELL_MARKER)
}

func (r *renderer) TableCell(out *bytes.Buffer, text []byte, flags int) {
	out.Write(text)
	out.WriteByte(_TABLE_CELL_MARKER)
}

func (r *renderer) Footnotes(out *bytes.Buffer, text func() bool) {
	marker := out.Len()
	r.ensureBlankLine(out)
	if !text() {
		out.Truncate(marker)
		return
	}
}

func (r *renderer) FootnoteItem(
	out *bytes.Buffer, name, text []byte, flags int) {
	out.Write(text)
	out.WriteByte('[')
	out.Write(name)
	out.WriteByte(']')
}

func (r *renderer) AutoLink(out *bytes.Buffer, link []byte, kind int) {
	if r.color {
		out.Write(ansiEscape.Blue)
	}
	out.Write(link)
	if r.color {
		out.Write(ansiEscape.Reset)
	}
}

func (r *renderer) CodeSpan(out *bytes.Buffer, text []byte) {
	if r.color {
		out.Write(ansiEscape.Green)
	} else {
		out.WriteByte('"')
	}
	out.Write(bytes.Replace(text, []byte(" "), []byte{_NBSP_MARKER}, -1))
	if r.color {
		out.Write(ansiEscape.Reset)
	} else {
		out.WriteByte('"')
	}
}

func (r *renderer) DoubleEmphasis(out *bytes.Buffer, text []byte) {
	if r.color {
		out.Write(ansiEscape.Bold)
	} else {
		out.WriteString("**")
	}
	out.Write(text)
	if r.color {
		out.Write(ansiEscape.Reset)
	} else {
		out.WriteString("**")
	}
}

func (r *renderer) Emphasis(out *bytes.Buffer, text []byte) {
	if r.color {
		out.Write(ansiEscape.Yellow)
	} else {
		out.WriteByte('*')
	}
	out.Write(text)
	if r.color {
		out.Write(ansiEscape.Reset)
	} else {
		out.WriteByte('*')
	}
}

func (r *renderer) Image(
	out *bytes.Buffer, link []byte, title []byte, alt []byte) {
	if r.color {
		out.Write(ansiEscape.Magenta)
	}
	if len(alt) > 0 {
		out.WriteByte('[')
		out.Write(alt)
		out.WriteByte(']')
		out.WriteByte(' ')
	} else if len(title) > 0 {
		out.WriteByte('[')
		out.Write(title)
		out.WriteByte(']')
		out.WriteByte(' ')
	}
	out.Write(link)
	if r.color {
		out.Write(ansiEscape.Reset)
	}
}

func (r *renderer) LineBreak(out *bytes.Buffer) {
	out.WriteByte(_LINE_BREAK_MARKER)
}

func (r *renderer) Link(
	out *bytes.Buffer, link []byte, title []byte, content []byte) {
	if r.color {
		out.Write(ansiEscape.Blue)
	}
	if len(content) > 0 && !bytes.Equal(content, link) {
		out.WriteByte('[')
		out.Write(content)
		out.WriteByte(']')
		out.WriteByte(' ')
	} else if len(title) > 0 && !bytes.Equal(title, link) {
		out.WriteByte('[')
		out.Write(title)
		out.WriteByte(']')
		out.WriteByte(' ')
	}
	out.Write(link)
	if r.color {
		out.Write(ansiEscape.Reset)
	}
}

func (r *renderer) RawHtmlTag(out *bytes.Buffer, tag []byte) {
	out.Write(tag)
}

func (r *renderer) TripleEmphasis(out *bytes.Buffer, text []byte) {
	if r.color {
		out.Write(ansiEscape.Bold)
		out.Write(ansiEscape.Red)
	} else {
		out.WriteString("***")
	}
	out.Write(text)
	if r.color {
		out.Write(ansiEscape.Reset)
	} else {
		out.WriteString("***")
	}
}

func (r *renderer) StrikeThrough(out *bytes.Buffer, text []byte) {
	if r.color {
		out.Write(ansiEscape.White)
	} else {
		out.WriteString("~~")
	}
	out.Write(text)
	if r.color {
		out.Write(ansiEscape.Reset)
	} else {
		out.WriteString("~~")
	}
}

func (r *renderer) FootnoteRef(out *bytes.Buffer, ref []byte, id int) {
	out.Write(ref)
	out.WriteString(" [")
	out.WriteString(fmt.Sprintf("%d", id))
	out.WriteByte(']')
}

func (r *renderer) Entity(out *bytes.Buffer, entity []byte) {
	out.Write(entity)
}

func (r *renderer) NormalText(out *bytes.Buffer, text []byte) {
	out.Write(text)
}

func (r *renderer) DocumentHeader(out *bytes.Buffer) {
}

func (r *renderer) DocumentFooter(out *bytes.Buffer) {
}

func (r *renderer) GetFlags() int {
	return 0
}

func (r *renderer) ensureNewLine(out *bytes.Buffer) {
	outb := out.Bytes()
	outbl := len(outb)
	if outbl > 0 && outb[outbl-1] != _LINE_BREAK_MARKER &&
		outb[outbl-1] != _INDENT_SUBSEQUENT_MARKER &&
		outb[outbl-1] != _INDENT_STOP_MARKER {
		out.WriteByte(_LINE_BREAK_MARKER)
	}
}

func (r *renderer) ensureBlankLine(out *bytes.Buffer) {
	outb := out.Bytes()
	outbl := len(outb)
	if outbl == 1 {
		if outb[0] != _LINE_BREAK_MARKER &&
			outb[0] != _INDENT_SUBSEQUENT_MARKER &&
			outb[0] != _INDENT_STOP_MARKER {
			out.WriteByte(_LINE_BREAK_MARKER)
			out.WriteByte(_LINE_BREAK_MARKER)
		} else {
			out.WriteByte(_LINE_BREAK_MARKER)
		}
	} else if outbl > 1 {
		if outb[outbl-1] != _LINE_BREAK_MARKER &&
			outb[outbl-1] != _INDENT_SUBSEQUENT_MARKER &&
			outb[outbl-1] != _INDENT_STOP_MARKER {
			out.WriteByte(_LINE_BREAK_MARKER)
			out.WriteByte(_LINE_BREAK_MARKER)
		} else if outb[outbl-2] != _LINE_BREAK_MARKER &&
			outb[outbl-2] != _INDENT_SUBSEQUENT_MARKER &&
			outb[outbl-2] != _INDENT_STOP_MARKER {
			out.WriteByte(_LINE_BREAK_MARKER)
		}
	}
}

func reflow(text []byte, indent1 []byte, indent2 []byte, width int) []byte {
	var out bytes.Buffer
	for {
		start := bytes.IndexByte(text, _INDENT_START_MARKER)
		if start >= 0 {
			out.Write(wrapBytes(text[:start], indent1, indent2, width))
			if out.Len() > 0 {
				indent1 = indent2
			}
			text = text[start+1:]
			nested := 1
			stop := bytes.IndexByte(text, _INDENT_STOP_MARKER)
			start = bytes.IndexByte(text, _INDENT_START_MARKER)
			for {
				if start == -1 || stop < start {
					nested--
					if nested == 0 {
						break
					}
					start = stop
				} else {
					nested++
				}
				if start == -1 {
					start = stop + 1
				}
				nextStop := bytes.IndexByte(
					text[start+1:], _INDENT_STOP_MARKER)
				if nextStop > -1 {
					nextStop += start + 1
				}
				stop = nextStop
				nextStart := bytes.IndexByte(
					text[start+1:], _INDENT_START_MARKER)
				if nextStart > -1 {
					nextStart += start + 1
				}
				start = nextStart
			}
			innerRawText := text[:stop]
			text = text[stop+1:]
			indentFirstMarker := bytes.IndexByte(
				innerRawText, _INDENT_FIRST_MARKER)
			innerFirstIndent := bytes.Join(
				[][]byte{indent1, innerRawText[:indentFirstMarker]}, []byte{})
			innerRawText = innerRawText[indentFirstMarker+1:]
			indentSubsequentMarker := bytes.IndexByte(
				innerRawText, _INDENT_SUBSEQUENT_MARKER)
			innerSubsequentIndent := bytes.Join(
				[][]byte{indent2, innerRawText[:indentSubsequentMarker]},
				[]byte{})
			innerRawText = innerRawText[indentSubsequentMarker+1:]
			out.Write(reflow(
				innerRawText, innerFirstIndent, innerSubsequentIndent, width))
			if out.Len() > 0 {
				indent1 = indent2
			}
		} else {
			out.Write(wrapBytes(text, indent1, indent2, width))
			if out.Len() > 0 {
				indent1 = indent2
			}
			break
		}
	}
	return out.Bytes()
}

func WrapBytes(text []byte, indent1 []byte, indent2 []byte, width int) []byte {
	if width == 0 {
		width = GetWidth()
	}
	text = wrapBytes(text, indent1, indent2, width)
	text = bytes.Replace(text, []byte{_LINE_BREAK_MARKER}, []byte{'\n'}, -1)
	return bytes.Trim(text, "\n")
}

func wrapBytes(text []byte, indent1 []byte, indent2 []byte, width int) []byte {
	if len(text) == 0 {
		return text
	}
	textLen := len(text)
	if textLen > 0 && text[textLen-1] == _LINE_BREAK_MARKER {
		text = text[:textLen-1]
	}
	var out bytes.Buffer
	for _, line := range bytes.Split(text, []byte{_LINE_BREAK_MARKER}) {
		if len(line) == 2 && line[0] == _HRULE_MARKER {
			var subout bytes.Buffer
			subout.Write(indent1)
			for subout.Len() < width {
				subout.WriteByte(line[1])
			}
			out.Write(subout.Bytes())
			out.WriteByte(_LINE_BREAK_MARKER)
			continue
		}
		lineLen := 0
		start := true
		for _, word := range bytes.Split(line, []byte{' '}) {
			wordLen := len(word)
			if wordLen == 0 {
				continue
			}
			scan := word
			for len(scan) > 1 {
				i := bytes.IndexByte(scan, '\x1b')
				if i == -1 {
					break
				}
				j := bytes.IndexByte(scan[i+1:], 'm')
				if j == -1 {
					i++
				} else {
					j += 2
					wordLen -= j
					scan = scan[i+j:]
				}
			}
			if start {
				if out.Len() == 0 {
					out.Write(indent1)
					lineLen += len(indent1)
				} else {
					out.Write(indent2)
					lineLen += len(indent2)
				}
				out.Write(word)
				lineLen += wordLen
				start = false
			} else if lineLen+1+wordLen >= width {
				out.WriteByte(_LINE_BREAK_MARKER)
				out.Write(indent2)
				out.Write(word)
				lineLen = len(indent2) + wordLen
			} else {
				out.WriteByte(' ')
				out.Write(word)
				lineLen += 1 + wordLen
			}
		}
		out.WriteByte(_LINE_BREAK_MARKER)
	}
	return out.Bytes()
}
