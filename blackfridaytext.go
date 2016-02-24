// Copyright Gregory Holt. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package blackfridaytext contains a text renderer for the
// Blackfriday Markdown Processor http://github.com/russross/blackfriday.
package blackfridaytext

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/gholt/brimtext"
	"github.com/russross/blackfriday"
)

type Options struct {
	// Width indicates how long to attempt to restrict lines to and may be a
	// positive int for a specific width, 0 for the default width (attempted to
	// get from terminal, 79 otherwise), or a negative number for a width
	// relative to the default.
	Width int
	// Color set true will allow ANSI Color Escape Codes.
	Color bool
	// Indent1 is the prefix for the first line.
	Indent1 string
	// Indent2 is the prefix for any second or subsequent lines.
	Indent2           string
	TableAlignOptions *brimtext.AlignOptions
}

func parseOptions(opt *Options) (int, bool, []byte, []byte, *brimtext.AlignOptions) {
	var width int
	var color bool
	var indent1 []byte
	var indent2 []byte
	var tableAlignOptions *brimtext.AlignOptions
	if opt != nil {
		width = opt.Width
		color = opt.Color
		indent1 = []byte(opt.Indent1)
		indent2 = []byte(opt.Indent2)
		tableAlignOptions = opt.TableAlignOptions
	}
	if width < 1 {
		width = brimtext.GetTTYWidth() + width
	}
	if tableAlignOptions == nil {
		tableAlignOptions = brimtext.NewSimpleAlignOptions()
	}
	return width, color, indent1, indent2, tableAlignOptions
}

const (
	_               byte = iota // 0 NUL
	markLineBreak               // 1 SOH
	markNBSP                    // 2 STX
	markIndentStart             // 3 ETX
	markIndent1                 // 4 EOT
	markIndent2                 // 5 ENQ
	markIndentStop              // 6 ACK
	markTableRow                // 7 BEL
	markTableCell               // 8 BS
	_                           // 9 TAB
	_                           // 10 LF
	markHRule                   // 11 VT
)

// MarkdownToText parses the markdown using the Blackfriday Markdown Processor
// and an internal renderer to return any metadata and the formatted text. If
// opt is nil the defaults will be used.
//
// See MarkdownMetadata for a description of the [][]string metadata returned.
func MarkdownToText(markdown []byte, opt *Options) ([][]string, []byte) {
	metadata, position := MarkdownMetadata(markdown)
	return metadata, MarkdownToTextNoMetadata(markdown[position:], opt)
}

// MarkdownMetadata parses just the metadata from the markdown and returns the
// metadata and the position of the rest of the markdown.
//
// The metadata is a [][]string where each []string will have two elements, the
// metadata item name and the value. Metadata is an extension of standard
// Markdown and is documented at
// https://github.com/fletcher/MultiMarkdown/wiki/MultiMarkdown-Syntax-Guide#metadata
// -- this implementation currently differs in that it does not support
// multiline values.
//
// In addition, the rest of markdown is scanned for lines containing only
// "///".
//
// If there is one "///" line, the text above that mark is considered the
// "Summary" metadata item; the summary will also be treated as part of the
// content (with the "///" line omitted). This is known as a "soft break".
//
// If there are two "///" lines, one right after the other, the summary will
// only be contained in the "Summary" metadata item and not part of the main
// content. This is known as a "hard break".
func MarkdownMetadata(markdown []byte) ([][]string, int) {
	metadata := make([][]string, 0)
	pos := 0
	for _, line := range bytes.Split(markdown, []byte("\n")) {
		sline := strings.Trim(string(line), " ")
		if sline == "" {
			break
		}
		colon := strings.Index(sline, ": ")
		if colon == -1 {
			// Since there's no blank line separating the metadata and content,
			// we assume there wasn't actually any metadata.
			metadata = make([][]string, 0)
			pos = 0
			break
		}
		name := strings.Trim(sline[:colon], " ")
		value := strings.Trim(sline[colon+1:], " ")
		metadata = append(metadata, []string{name, value})
		pos += len(line) + 1
	}
	if pos > len(markdown) {
		pos = len(markdown) - 1
	}
	pos2 := bytes.Index(markdown[pos:], []byte("\n///\n"))
	if pos2 != -1 {
		value := string(markdown[pos : pos+pos2])
		metadata = append(metadata, []string{"Summary", value})
		if string(markdown[pos+pos2+5:pos+pos2+9]) == "///\n" {
			pos += pos2 + 9
		}
	}
	return metadata, pos
}

// MarkdownToTextNoMetadata is the same as MarkdownToText only skipping the
// detection and parsing of any leading metadata. If opt is nil the defaults
// will be used.
func MarkdownToTextNoMetadata(markdown []byte, opt *Options) []byte {
	width, color, indent1, indent2, tableAlignOptions := parseOptions(opt)
	rend := &renderer{
		width:             width,
		color:             color,
		tableAlignOptions: tableAlignOptions,
	}
	markdown = bytes.Replace(markdown, []byte("\n///\n"), []byte(""), -1)
	txt := blackfriday.Markdown(markdown, rend,
		blackfriday.EXTENSION_NO_INTRA_EMPHASIS|
			blackfriday.EXTENSION_TABLES|
			blackfriday.EXTENSION_FENCED_CODE|
			blackfriday.EXTENSION_AUTOLINK|
			blackfriday.EXTENSION_STRIKETHROUGH)
	for rend.level > 0 {
		txt = append(txt, markIndentStop)
		rend.level--
	}
	if len(txt) > 0 {
		txt = bytes.Replace(txt, []byte(" \n"), []byte(" "), -1)
		txt = bytes.Replace(txt, []byte("\n"), []byte(" "), -1)
		txt = reflow(txt, indent1, indent2, rend.width)
		txt = bytes.Replace(txt, []byte{markNBSP}, []byte(" "), -1)
		txt = bytes.Replace(txt, []byte{markLineBreak}, []byte("\n"), -1)
	}
	return txt
}

type renderer struct {
	width             int
	color             bool
	tableAlignOptions *brimtext.AlignOptions
	level             int
}

func (rend *renderer) BlockCode(out *bytes.Buffer, text []byte, lang string) {
	length := len(text)
	if length > 0 && text[length-1] == '\n' {
		text = text[:length-1]
	}
	rend.ensureBlankLine(out)
	for _, line := range bytes.Split(text, []byte("\n")) {
		if rend.color {
			out.Write(brimtext.ANSIEscape.FGreen)
		}
		out.Write(bytes.Replace(line, []byte(" "), []byte{markNBSP}, -1))
		if rend.color {
			out.Write(brimtext.ANSIEscape.Reset)
		}
		out.WriteByte(markLineBreak)
	}
	rend.ensureBlankLine(out)
}

func (rend *renderer) BlockQuote(out *bytes.Buffer, text []byte) {
	rend.ensureBlankLine(out)
	out.WriteByte(markIndentStart)
	out.WriteString("> ")
	out.WriteByte(markIndent1)
	out.WriteString("> ")
	out.WriteByte(markIndent2)
	out.Write(bytes.Trim(text, string([]byte{markLineBreak})))
	out.WriteByte(markIndentStop)
}

func (rend *renderer) BlockHtml(out *bytes.Buffer, text []byte) {
	rend.ensureBlankLine(out)
	out.Write(bytes.Replace(text, []byte("\n"), []byte{markLineBreak}, -1))
}

func (rend *renderer) Header(out *bytes.Buffer, text func() bool, level int, id string) {
	oPos := out.Len()
	rend.ensureBlankLine(out)
	oLevel := rend.level
	level--
	for rend.level > level {
		out.WriteByte(markIndentStop)
		rend.level--
	}
	out.WriteByte(markIndentStart)
	out.WriteString("--[ ")
	out.WriteByte(markIndent1)
	out.WriteString("    ")
	out.WriteByte(markIndent2)
	if rend.color {
		out.Write(brimtext.ANSIEscape.Bold)
	}
	if !text() {
		out.Truncate(oPos)
		rend.level = oLevel
		return
	}
	if rend.color {
		out.Write(brimtext.ANSIEscape.Reset)
	}
	out.WriteByte(markNBSP)
	out.WriteString("]--")
	out.WriteByte(markIndentStop)
	for rend.level <= level {
		out.WriteByte(markIndentStart)
		out.WriteString("    ")
		out.WriteByte(markIndent1)
		out.WriteString("    ")
		out.WriteByte(markIndent2)
		rend.level++
	}
	rend.ensureBlankLine(out)
}

func (rend *renderer) HRule(out *bytes.Buffer) {
	rend.ensureBlankLine(out)
	out.WriteByte(markHRule)
	out.WriteByte('-')
	rend.ensureBlankLine(out)
}

func (rend *renderer) List(out *bytes.Buffer, text func() bool, flags int) {
	oPos := out.Len()
	rend.ensureNewLine(out)
	if !text() {
		out.Truncate(oPos)
		return
	}
}

func (rend *renderer) ListItem(out *bytes.Buffer, text []byte, flags int) {
	rend.ensureNewLine(out)
	out.WriteByte(markIndentStart)
	out.WriteString("  * ")
	out.WriteByte(markIndent1)
	out.WriteString("    ")
	out.WriteByte(markIndent2)
	out.Write(bytes.Trim(text, string([]byte{markLineBreak})))
	out.WriteByte(markIndentStop)
}

func (rend *renderer) Paragraph(out *bytes.Buffer, text func() bool) {
	rend.ensureBlankLine(out)
	oPos := out.Len()
	if !text() {
		out.Truncate(oPos)
	}
}

func (rend *renderer) Table(out *bytes.Buffer, header []byte, body []byte, columnData []int) {
	rend.ensureBlankLine(out)
	data := make([][]string, 0)
	rows := bytes.Split(header[:len(header)-1], []byte{markTableRow})
	for _, row := range rows {
		headerRow := make([]string, 0)
		cells := bytes.Split(row[:len(row)-1], []byte{markTableCell})
		for _, cell := range cells {
			headerRow = append(headerRow, string(cell))
		}
		if len(headerRow) > 0 && headerRow[0] != "omit" {
			data = append(data, headerRow)
		}
	}
	if len(data) > 0 {
		data = append(data, nil)
	}
	rows = bytes.Split(body[:len(body)-1], []byte{markTableRow})
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		bodyRow := make([]string, 0)
		cells := bytes.Split(row[:len(row)-1], []byte{markTableCell})
		for _, cell := range cells {
			bodyRow = append(bodyRow, string(cell))
		}
		data = append(data, bodyRow)
	}
	text := []byte(brimtext.Align(data, rend.tableAlignOptions))
	text = bytes.Replace(text, []byte{' '}, []byte{markNBSP}, -1)
	text = bytes.Replace(text, []byte{'\n'}, []byte{markLineBreak}, -1)
	out.Write(text)
}

func (rend *renderer) TableRow(out *bytes.Buffer, text []byte) {
	out.Write(text)
	out.WriteByte(markTableRow)
}

func (rend *renderer) TableHeaderCell(out *bytes.Buffer, text []byte, flags int) {
	out.Write(text)
	out.WriteByte(markTableCell)
}

func (rend *renderer) TableCell(out *bytes.Buffer, text []byte, flags int) {
	out.Write(text)
	out.WriteByte(markTableCell)
}

func (rend *renderer) Footnotes(out *bytes.Buffer, text func() bool) {
	oPos := out.Len()
	rend.ensureBlankLine(out)
	if !text() {
		out.Truncate(oPos)
		return
	}
}

func (rend *renderer) FootnoteItem(out *bytes.Buffer, name, text []byte, flags int) {
	out.Write(text)
	out.WriteByte('[')
	out.Write(name)
	out.WriteByte(']')
}

func (rend *renderer) TitleBlock(out *bytes.Buffer, text []byte) {
	text = bytes.TrimPrefix(text, []byte("% "))
	text = bytes.Replace(text, []byte("\n% "), []byte("\n"), -1)
	out.Write(text)
}

func (rend *renderer) AutoLink(out *bytes.Buffer, link []byte, kind int) {
	if rend.color {
		out.Write(brimtext.ANSIEscape.FBlue)
	}
	out.Write(link)
	if rend.color {
		out.Write(brimtext.ANSIEscape.Reset)
	}
}

func (rend *renderer) CodeSpan(out *bytes.Buffer, text []byte) {
	if rend.color {
		out.Write(brimtext.ANSIEscape.FGreen)
	} else {
		out.WriteByte('"')
	}
	out.Write(bytes.Replace(text, []byte(" "), []byte{markNBSP}, -1))
	if rend.color {
		out.Write(brimtext.ANSIEscape.Reset)
	} else {
		out.WriteByte('"')
	}
}

func (rend *renderer) DoubleEmphasis(out *bytes.Buffer, text []byte) {
	if rend.color {
		out.Write(brimtext.ANSIEscape.Bold)
	} else {
		out.WriteString("**")
	}
	out.Write(text)
	if rend.color {
		out.Write(brimtext.ANSIEscape.Reset)
	} else {
		out.WriteString("**")
	}
}

func (rend *renderer) Emphasis(out *bytes.Buffer, text []byte) {
	if rend.color {
		out.Write(brimtext.ANSIEscape.FYellow)
	} else {
		out.WriteByte('*')
	}
	out.Write(text)
	if rend.color {
		out.Write(brimtext.ANSIEscape.Reset)
	} else {
		out.WriteByte('*')
	}
}

func (rend *renderer) Image(out *bytes.Buffer, link []byte, title []byte, alt []byte) {
	if rend.color {
		out.Write(brimtext.ANSIEscape.FMagenta)
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
	if rend.color {
		out.Write(brimtext.ANSIEscape.Reset)
	}
}

func (rend *renderer) LineBreak(out *bytes.Buffer) {
	out.WriteByte(markLineBreak)
}

func (rend *renderer) Link(out *bytes.Buffer, link []byte, title []byte, content []byte) {
	if rend.color {
		out.Write(brimtext.ANSIEscape.FBlue)
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
	if rend.color {
		out.Write(brimtext.ANSIEscape.Reset)
	}
}

func (rend *renderer) RawHtmlTag(out *bytes.Buffer, tag []byte) {
	out.Write(tag)
}

func (rend *renderer) TripleEmphasis(out *bytes.Buffer, text []byte) {
	if rend.color {
		out.Write(brimtext.ANSIEscape.Bold)
		out.Write(brimtext.ANSIEscape.FRed)
	} else {
		out.WriteString("***")
	}
	out.Write(text)
	if rend.color {
		out.Write(brimtext.ANSIEscape.Reset)
	} else {
		out.WriteString("***")
	}
}

func (rend *renderer) StrikeThrough(out *bytes.Buffer, text []byte) {
	if rend.color {
		out.Write(brimtext.ANSIEscape.FWhite)
	} else {
		out.WriteString("~~")
	}
	out.Write(text)
	if rend.color {
		out.Write(brimtext.ANSIEscape.Reset)
	} else {
		out.WriteString("~~")
	}
}

func (rend *renderer) FootnoteRef(out *bytes.Buffer, ref []byte, id int) {
	out.Write(ref)
	out.WriteString(" [")
	out.WriteString(fmt.Sprintf("%d", id))
	out.WriteByte(']')
}

func (rend *renderer) Entity(out *bytes.Buffer, entity []byte) {
	out.Write(entity)
}

func (rend *renderer) NormalText(out *bytes.Buffer, text []byte) {
	out.Write(text)
}

func (rend *renderer) DocumentHeader(out *bytes.Buffer) {
}

func (rend *renderer) DocumentFooter(out *bytes.Buffer) {
}

func (rend *renderer) GetFlags() int {
	return 0
}

func (rend *renderer) ensureNewLine(out *bytes.Buffer) {
	bs := out.Bytes()
	bsLen := len(bs)
	if bsLen > 0 {
		last := bs[bsLen-1]
		if last != markLineBreak && last != markIndent2 && last != markIndentStop {
			out.WriteByte(markLineBreak)
		}
	}
}

func (rend *renderer) ensureBlankLine(out *bytes.Buffer) {
	bs := out.Bytes()
	bsLen := len(bs)
	if bsLen == 1 {
		first := bs[0]
		if first != markLineBreak && first != markIndent2 && first != markIndentStop {
			out.WriteByte(markLineBreak)
			out.WriteByte(markLineBreak)
		} else {
			out.WriteByte(markLineBreak)
		}
	} else if bsLen > 1 {
		last := bs[bsLen-1]
		if last != markLineBreak && last != markIndent2 && last != markIndentStop {
			out.WriteByte(markLineBreak)
			out.WriteByte(markLineBreak)
		} else {
			secondLast := bs[bsLen-2]
			if secondLast != markLineBreak && secondLast != markIndent2 && secondLast != markIndentStop {
				out.WriteByte(markLineBreak)
			}
		}
	}
}

func reflow(text []byte, indent1 []byte, indent2 []byte, width int) []byte {
	var out bytes.Buffer
	for {
		start := bytes.IndexByte(text, markIndentStart)
		if start >= 0 {
			out.Write(wrapBytes(text[:start], width, indent1, indent2))
			if out.Len() > 0 {
				indent1 = indent2
			}
			text = text[start+1:]
			nested := 1
			stop := bytes.IndexByte(text, markIndentStop)
			start = bytes.IndexByte(text, markIndentStart)
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
				nextStop := bytes.IndexByte(text[start+1:], markIndentStop)
				if nextStop > -1 {
					nextStop += start + 1
				}
				stop = nextStop
				nextStart := bytes.IndexByte(text[start+1:], markIndentStart)
				if nextStart > -1 {
					nextStart += start + 1
				}
				start = nextStart
			}
			innerRawText := text[:stop]
			text = text[stop+1:]
			indent1Marker := bytes.IndexByte(innerRawText, markIndent1)
			innerIndent1 := bytes.Join(
				[][]byte{indent1, innerRawText[:indent1Marker]}, []byte{})
			innerRawText = innerRawText[indent1Marker+1:]
			indent2Marker := bytes.IndexByte(innerRawText, markIndent2)
			innerIndent2 := bytes.Join(
				[][]byte{indent2, innerRawText[:indent2Marker]}, []byte{})
			innerRawText = innerRawText[indent2Marker+1:]
			out.Write(reflow(innerRawText, innerIndent1, innerIndent2, width))
			if out.Len() > 0 {
				indent1 = indent2
			}
		} else {
			out.Write(wrapBytes(text, width, indent1, indent2))
			if out.Len() > 0 {
				indent1 = indent2
			}
			break
		}
	}
	return out.Bytes()
}

func wrapBytes(text []byte, width int, indent1 []byte, indent2 []byte) []byte {
	if len(text) == 0 {
		return text
	}
	textLen := len(text)
	if textLen > 0 && text[textLen-1] == markLineBreak {
		text = text[:textLen-1]
	}
	var out bytes.Buffer
	for _, line := range bytes.Split(text, []byte{markLineBreak}) {
		if len(line) == 2 && line[0] == markHRule {
			var subout bytes.Buffer
			subout.Write(indent1)
			for subout.Len() < width {
				subout.WriteByte(line[1])
			}
			out.Write(subout.Bytes())
			out.WriteByte(markLineBreak)
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
				out.WriteByte(markLineBreak)
				out.Write(indent2)
				out.Write(word)
				lineLen = len(indent2) + wordLen
			} else {
				out.WriteByte(' ')
				out.Write(word)
				lineLen += 1 + wordLen
			}
		}
		out.WriteByte(markLineBreak)
	}
	return out.Bytes()
}
