package markdown

import (
	"bytes"
)

type smartypantsData struct {
	inSingleQuote bool
	inDoubleQuote bool
}

func wordBoundary(c byte) bool {
	return c == 0 || isspace(c) || ispunct(c)
}

func tolower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c - 'A' + 'a'
	}
	return c
}

func isdigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func smartQuoteHelper(out *bytes.Buffer, previousChar byte, nextChar byte, quote byte, isOpen *bool) bool {
	// edge of the buffer is likely to be a tag that we don't get to see,
	// so we treat it like text sometimes

	// enumerate all sixteen possibilities for (previousChar, nextChar)
	// each can be one of {0, space, punct, other}
	switch {
	case previousChar == 0 && nextChar == 0:
		// context is not any help here, so toggle
		*isOpen = !*isOpen
	case isspace(previousChar) && nextChar == 0:
		// [ "] might be [ "<code>foo...]
		*isOpen = true
	case ispunct(previousChar) && nextChar == 0:
		// [!"] hmm... could be [Run!"] or [("<code>...]
		*isOpen = false
	case /* isnormal(previousChar) && */ nextChar == 0:
		// [a"] is probably a close
		*isOpen = false
	case previousChar == 0 && isspace(nextChar):
		// [" ] might be [...foo</code>" ]
		*isOpen = false
	case isspace(previousChar) && isspace(nextChar):
		// [ " ] context is not any help here, so toggle
		*isOpen = !*isOpen
	case ispunct(previousChar) && isspace(nextChar):
		// [!" ] is probably a close
		*isOpen = false
	case /* isnormal(previousChar) && */ isspace(nextChar):
		// [a" ] this is one of the easy cases
		*isOpen = false
	case previousChar == 0 && ispunct(nextChar):
		// ["!] hmm... could be ["$1.95] or [</code>"!...]
		*isOpen = false
	case isspace(previousChar) && ispunct(nextChar):
		// [ "!] looks more like [ "$1.95]
		*isOpen = true
	case ispunct(previousChar) && ispunct(nextChar):
		// [!"!] context is not any help here, so toggle
		*isOpen = !*isOpen
	case /* isnormal(previousChar) && */ ispunct(nextChar):
		// [a"!] is probably a close
		*isOpen = false
	case previousChar == 0 /* && isnormal(nextChar) */ :
		// ["a] is probably an open
		*isOpen = true
	case isspace(previousChar) /* && isnormal(nextChar) */ :
		// [ "a] this is one of the easy cases
		*isOpen = true
	case ispunct(previousChar) /* && isnormal(nextChar) */ :
		// [!"a] is probably an open
		*isOpen = true
	default:
		// [a'b] maybe a contraction?
		*isOpen = false
	}

	out.WriteByte('&')
	if *isOpen {
		out.WriteByte('l')
	} else {
		out.WriteByte('r')
	}
	out.WriteByte(quote)
	out.WriteString("quo;")
	return true
}

func smartSingleQuote(out *bytes.Buffer, smrt *smartypantsData, previousChar byte, text []byte) int {
	if len(text) >= 2 {
		t1 := tolower(text[1])

		if t1 == '\'' {
			nextChar := byte(0)
			if len(text) >= 3 {
				nextChar = text[2]
			}
			if smartQuoteHelper(out, previousChar, nextChar, 'd', &smrt.inDoubleQuote) {
				return 1
			}
		}

		if (t1 == 's' || t1 == 't' || t1 == 'm' || t1 == 'd') && (len(text) < 3 || wordBoundary(text[2])) {
			out.WriteString("&rsquo;")
			return 0
		}

		if len(text) >= 3 {
			t2 := tolower(text[2])

			if ((t1 == 'r' && t2 == 'e') || (t1 == 'l' && t2 == 'l') || (t1 == 'v' && t2 == 'e')) &&
				(len(text) < 4 || wordBoundary(text[3])) {
				out.WriteString("&rsquo;")
				return 0
			}
		}
	}

	nextChar := byte(0)
	if len(text) > 1 {
		nextChar = text[1]
	}
	if smartQuoteHelper(out, previousChar, nextChar, 's', &smrt.inSingleQuote) {
		return 0
	}

	out.WriteByte(text[0])
	return 0
}

func smartParens(out *bytes.Buffer, smrt *smartypantsData, previousChar byte, text []byte) int {
	if len(text) >= 3 {
		t1 := tolower(text[1])
		t2 := tolower(text[2])

		if t1 == 'c' && t2 == ')' {
			out.WriteRune('©') //&copy;
			return 2
		}

		if t1 == 'r' && t2 == ')' {
			out.WriteRune('®') // &reg;
			return 2
		}

		if len(text) >= 4 && t1 == 't' && t2 == 'm' && text[3] == ')' {
			out.WriteRune('™') // &trade;
			return 3
		}
	}

	out.WriteByte(text[0])
	return 0
}

func smartDash(out *bytes.Buffer, smrt *smartypantsData, previousChar byte, text []byte) int {
	if len(text) >= 2 {
		if text[1] == '-' {
			out.WriteRune('—') // &mdash;
			return 1
		}

		if wordBoundary(previousChar) && wordBoundary(text[1]) {
			out.WriteRune('–') // &ndash;
			return 0
		}
	}

	out.WriteByte(text[0])
	return 0
}

func smartAmp(out *bytes.Buffer, smrt *smartypantsData, previousChar byte, text []byte) int {
	if bytes.HasPrefix(text, []byte("&quot;")) {
		nextChar := byte(0)
		if len(text) >= 7 {
			nextChar = text[6]
		}
		if smartQuoteHelper(out, previousChar, nextChar, 'd', &smrt.inDoubleQuote) {
			return 5
		}
	}

	if bytes.HasPrefix(text, []byte("&#0;")) {
		return 3
	}

	out.WriteByte('&')
	return 0
}

func smartPeriod(out *bytes.Buffer, smrt *smartypantsData, previousChar byte, text []byte) int {
	if len(text) >= 3 && text[1] == '.' && text[2] == '.' {
		out.WriteRune('…') // &hellip;
		return 2
	}

	if len(text) >= 5 && text[1] == ' ' && text[2] == '.' && text[3] == ' ' && text[4] == '.' {
		out.WriteRune('…') // &hellip;
		return 4
	}

	out.WriteByte(text[0])
	return 0
}

func smartBacktick(out *bytes.Buffer, smrt *smartypantsData, previousChar byte, text []byte) int {
	if len(text) >= 2 && text[1] == '`' {
		nextChar := byte(0)
		if len(text) >= 3 {
			nextChar = text[2]
		}
		if smartQuoteHelper(out, previousChar, nextChar, 'd', &smrt.inDoubleQuote) {
			return 1
		}
	}

	out.WriteByte(text[0])
	return 0
}

func smartNumberGeneric(out *bytes.Buffer, smrt *smartypantsData, previousChar byte, text []byte) int {
	if wordBoundary(previousChar) && len(text) >= 3 {
		// is it of the form digits/digits(word boundary)?, i.e., \d+/\d+\b
		// note: check for regular slash (/) or fraction slash (⁄, 0x2044, or 0xe2 81 84 in utf-8)
		numEnd := 0
		for len(text) > numEnd && isdigit(text[numEnd]) {
			numEnd++
		}
		if numEnd == 0 {
			out.WriteByte(text[0])
			return 0
		}
		denStart := numEnd + 1
		if len(text) > numEnd+3 && text[numEnd] == 0xe2 && text[numEnd+1] == 0x81 && text[numEnd+2] == 0x84 {
			denStart = numEnd + 3
		} else if len(text) < numEnd+2 || text[numEnd] != '/' {
			out.WriteByte(text[0])
			return 0
		}
		denEnd := denStart
		for len(text) > denEnd && isdigit(text[denEnd]) {
			denEnd++
		}
		if denEnd == denStart {
			out.WriteByte(text[0])
			return 0
		}
		if len(text) == denEnd || wordBoundary(text[denEnd]) {
			out.WriteString("<sup>")
			out.Write(text[:numEnd])
			out.WriteString("</sup>&frasl;<sub>")
			out.Write(text[denStart:denEnd])
			out.WriteString("</sub>")
			return denEnd - 1
		}
	}

	out.WriteByte(text[0])
	return 0
}

func smartDoubleQuote(out *bytes.Buffer, smrt *smartypantsData, previousChar byte, text []byte) int {
	nextChar := byte(0)
	if len(text) > 1 {
		nextChar = text[1]
	}
	if !smartQuoteHelper(out, previousChar, nextChar, 'd', &smrt.inDoubleQuote) {
		out.WriteString("&quot;")
	}

	return 0
}

func smartLeftAngle(out *bytes.Buffer, smrt *smartypantsData, previousChar byte, text []byte) int {
	i := 0
	for i < len(text) && text[i] != '>' {
		i++
	}
	out.Write(text[:i+1])
	return i
}

type smartCallback func(out *bytes.Buffer, smrt *smartypantsData, previousChar byte, text []byte) int

type smartypantsRenderer [256]smartCallback

func smartypants() *smartypantsRenderer {
	r := new(smartypantsRenderer)
	r['"'] = smartDoubleQuote
	r['&'] = smartAmp
	r['\''] = smartSingleQuote
	r['('] = smartParens
	r['-'] = smartDash
	r['.'] = smartPeriod
	for ch := '1'; ch <= '9'; ch++ {
		r[ch] = smartNumberGeneric
	}
	r['<'] = smartLeftAngle
	r['`'] = smartBacktick
	return r
}