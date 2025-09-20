//go:build test

package renderers

import (
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

func TestCustomImageRenderer(t *testing.T) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.CJK,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)
	md.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewCustomImageRenderer(), 100),
	))

	input := `![alt text](https://example.com/image.png "Title text")`
	var buf strings.Builder
	if err := md.Convert([]byte(input), &buf); err != nil {
		t.Fatalf("Markdown conversion failed: %v", err)
	}
	output := buf.String()

	// Check for custom renderer output (should be a link, not <img>)
	if !strings.Contains(output, `<a href="https://example.com/image.png"`) {
		t.Errorf("Custom image renderer not used. Output: %s", output)
	}
	if strings.Contains(output, "<img") {
		t.Errorf("Standard <img> tag found, custom renderer not applied. Output: %s", output)
	}
}
