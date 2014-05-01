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
	"fmt"
	"github.com/russross/blackfriday"
	"strings"
)

const (
	_BLACKFRIDAY_EXTENSIONS = blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
		blackfriday.EXTENSION_TABLES | blackfriday.EXTENSION_FENCED_CODE |
		blackfriday.EXTENSION_STRIKETHROUGH
)

const (
	_ byte = iota
	_LINE_BREAK_MARKER
	_NBSP_MARKER
	_INDENT_START_MARKER
	_INDENT_FIRST_MARKER
	_INDENT_SUBSEQUENT_MARKER
	_INDENT_STOP_MARKER
	_TABLE_ROW_MARKER
	_TABLE_CELL_MARKER
)

// MarkdownToText parses the markdown text using the Blackfriday Markdown
// Processor and an internal renderer to return any metadata and the formatted
// text.
//
// The metadata is a [][]string where each []string will have two elements, the
// metadata item name and the value. Metadata is an extension of standard
// Markdown and is documented at
// https://github.com/fletcher/MultiMarkdown/wiki/MultiMarkdown-Syntax-Guide#metadata
// -- this implementation currently differs in that it requires the trailing
// two spaces on each metadata line and doesn't support multiline values.
func MarkdownToText(markdown []byte) ([][]string, []byte) {
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
	if len(text) > 0 {
		text = blackfriday.Markdown(
			markdown[position:], &renderer{}, _BLACKFRIDAY_EXTENSIONS)
		if len(text) > 0 {
			text = bytes.Replace(text, []byte(" \n"), []byte(" "), -1)
			text = bytes.Replace(text, []byte("\n"), []byte(" "), -1)
			text = reflow(text, []byte{}, []byte{})
			text = bytes.Replace(text, []byte{_NBSP_MARKER}, []byte(" "), -1)
			text = bytes.Replace(
				text, []byte{_LINE_BREAK_MARKER}, []byte("\n"), -1)
		}
	}
	return metadata, text
}

type renderer struct {
}

func (r *renderer) BlockCode(out *bytes.Buffer, text []byte, lang string) {
	r.ensureBlankLine(out)
	text = bytes.Replace(text, []byte("\n"), []byte{_LINE_BREAK_MARKER}, -1)
	text = bytes.Replace(text, []byte(" "), []byte{_NBSP_MARKER}, -1)
	out.Write(text)
	r.ensureNewLine(out)
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
	r.ensureNewLine(out)
}

func (r *renderer) BlockHtml(out *bytes.Buffer, text []byte) {
	r.ensureBlankLine(out)
	out.Write(bytes.Replace(
		text, []byte("\n"), []byte{_LINE_BREAK_MARKER}, -1))
	r.ensureNewLine(out)
}

func (r *renderer) Header(
	out *bytes.Buffer, text func() bool, level int, id string) {
	marker := out.Len()
	r.ensureBlankLine(out)
	for i := 0; i < level; i++ {
		out.WriteByte('#')
	}
	out.WriteByte(' ')
	if !text() {
		out.Truncate(marker)
		return
	}
	r.ensureNewLine(out)
}

func (r *renderer) HRule(out *bytes.Buffer) {
	r.ensureBlankLine(out)
	for i := 79; i > 0; i-- {
		out.WriteByte('-')
	}
	r.ensureNewLine(out)
}

func (r *renderer) List(out *bytes.Buffer, text func() bool, flags int) {
	marker := out.Len()
	r.ensureBlankLine(out)
	if !text() {
		out.Truncate(marker)
		return
	}
}

func (r *renderer) ListItem(out *bytes.Buffer, text []byte, flags int) {
	r.ensureNewLine(out)
	out.WriteString("  * ")
	out.Write(text)
	r.ensureNewLine(out)
}

func (r *renderer) Paragraph(out *bytes.Buffer, text func() bool) {
	r.ensureBlankLine(out)
	marker := out.Len()
	if !text() {
		out.Truncate(marker)
	}
	r.ensureNewLine(out)
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
	r.ensureNewLine(out)
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
	out.Write(link)
}

func (r *renderer) CodeSpan(out *bytes.Buffer, text []byte) {
	out.WriteByte('"')
	out.Write(bytes.Replace(text, []byte(" "), []byte{_NBSP_MARKER}, -1))
	out.WriteByte('"')
}

func (r *renderer) DoubleEmphasis(out *bytes.Buffer, text []byte) {
	out.WriteString("**")
	out.Write(text)
	out.WriteString("**")
}

func (r *renderer) Emphasis(out *bytes.Buffer, text []byte) {
	out.WriteByte('*')
	out.Write(text)
	out.WriteByte('*')
}

func (r *renderer) Image(
	out *bytes.Buffer, link []byte, title []byte, alt []byte) {
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
}

func (r *renderer) LineBreak(out *bytes.Buffer) {
	out.WriteByte(_LINE_BREAK_MARKER)
}

func (r *renderer) Link(
	out *bytes.Buffer, link []byte, title []byte, content []byte) {
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
}

func (r *renderer) RawHtmlTag(out *bytes.Buffer, tag []byte) {
	out.Write(tag)
}

func (r *renderer) TripleEmphasis(out *bytes.Buffer, text []byte) {
	out.WriteString("***")
	out.Write(text)
	out.WriteString("***")
}

func (r *renderer) StrikeThrough(out *bytes.Buffer, text []byte) {
	out.WriteString("~~")
	out.Write(text)
	out.WriteString("~~")
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
		outb[outbl-1] != _INDENT_STOP_MARKER {
		out.WriteByte(_LINE_BREAK_MARKER)
	}
}

func (r *renderer) ensureBlankLine(out *bytes.Buffer) {
	outb := out.Bytes()
	outbl := len(outb)
	if outbl == 1 {
		if outb[0] != _LINE_BREAK_MARKER && outb[0] != _INDENT_STOP_MARKER {
			out.WriteByte(_LINE_BREAK_MARKER)
			out.WriteByte(_LINE_BREAK_MARKER)
		} else {
			out.WriteByte(_LINE_BREAK_MARKER)
		}
	} else if outbl > 1 {
		if outb[outbl-1] != _LINE_BREAK_MARKER &&
			outb[outbl-1] != _INDENT_STOP_MARKER {
			out.WriteByte(_LINE_BREAK_MARKER)
			out.WriteByte(_LINE_BREAK_MARKER)
		} else if outb[outbl-2] != _LINE_BREAK_MARKER &&
			outb[outbl-2] != _INDENT_STOP_MARKER {
			out.WriteByte(_LINE_BREAK_MARKER)
		}
	}
}

func reflow(text []byte, indent1 []byte, indent2 []byte) []byte {
	var out bytes.Buffer
	for {
		start := bytes.IndexByte(text, _INDENT_START_MARKER)
		if start >= 0 {
			out.Write(wrap(text[:start], indent1, indent2))
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
				innerRawText, innerFirstIndent, innerSubsequentIndent))
			if out.Len() > 0 {
				indent1 = indent2
			}
		} else {
			out.Write(wrap(text, indent1, indent2))
			if out.Len() > 0 {
				indent1 = indent2
			}
			break
		}
	}
	return out.Bytes()
}

func wrap(text []byte, indent1 []byte, indent2 []byte) []byte {
	var out bytes.Buffer
	rawTextLen := len(text)
	if rawTextLen > 0 && text[rawTextLen-1] == _LINE_BREAK_MARKER {
		text = text[:rawTextLen-1]
	}
	for _, line := range bytes.Split(text, []byte{_LINE_BREAK_MARKER}) {
		sline := strings.Trim(string(line), " ")
		for strings.Index(sline, "  ") != -1 {
			sline = strings.Replace(sline, "  ", " ", -1)
		}
		if out.Len() == 0 {
			sline = string(indent1) + sline
		} else {
			sline = string(indent2) + sline
		}
		for len(sline) > 79 {
			index := strings.LastIndex(sline[:79], " ")
			if index == -1 {
				index = strings.Index(sline[79:], " ")
				if index == -1 {
					break
				}
				index += 79
			}
			out.Write([]byte(sline[:index]))
			out.WriteByte(_LINE_BREAK_MARKER)
			out.Write(indent2)
			sline = sline[index+1:]
		}
		out.Write([]byte(sline))
		out.WriteByte(_LINE_BREAK_MARKER)
	}
	return out.Bytes()
}
