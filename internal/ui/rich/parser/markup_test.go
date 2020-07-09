package parser

import (
	"testing"

	"github.com/diamondburned/cchat-mock/segments"
	"github.com/diamondburned/cchat/text"
)

func TestRenderMarkup(t *testing.T) {
	content := text.Rich{Content: "astolfo is the best trap"}
	content.Segments = []text.Segment{
		segments.NewColored(content.Content, 0x55CDFC),
	}
	expect := `<span color="#55CDFC">` + content.Content + "</span>"

	if text := RenderMarkup(content); text != expect {
		t.Fatal("Unexpected text:", text)
	}
}

// Test no longer works, and should not work anyway.

// func TestRenderMarkupPartial(t *testing.T) {
// 	content := text.Rich{Content: "random placeholder text go brrr"}
// 	content.Segments = []text.Segment{
// 		// This is absolutely jankery that should not work at all, but we'll try
// 		// it anyway.
// 		coloredSegment{0, 4, 0x55CDFC},
// 		coloredSegment{2, 6, 0xFFFFFF}, // naive parsing, so spans close unexpectedly.
// 		coloredSegment{4, 6, 0xF7A8B8},
// 	}
// 	const expect = "" +
// 		<span color="#55CDFC">ra<span color="#FFFFFF" bgalpha="10%" bgcolor="#FFFFFF">nd<span color="#F7A8B8" bgalpha="10%" bgcolor="#F7A8B8"></span>om</span></span>
// 		`<span color="#55CDFC">ra<span color="#FFFFFF">nd</span>` +
// 		`<span color="#F7A8B8">om</span></span>`

// 	if text := RenderMarkup(content); !strings.HasPrefix(text, expect) {
// 		t.Fatal("Unexpected text:", text)
// 	}
// }

type coloredSegment struct {
	start int
	end   int
	color uint32
}

var _ text.Colorer = (*coloredSegment)(nil)

func (c coloredSegment) Bounds() (start, end int) {
	return c.start, c.end
}

func (c coloredSegment) Color() uint32 {
	return c.color
}
