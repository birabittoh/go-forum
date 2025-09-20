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

func TestCustomLinkRenderer(t *testing.T) {
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
		util.Prioritized(NewCustomLinkRenderer(), 100),
	))

	input := `[alt text](https://example.com "Title")`
	var buf strings.Builder
	if err := md.Convert([]byte(input), &buf); err != nil {
		t.Fatalf("Markdown conversion failed: %v", err)
	}
	output := buf.String()

	// Check for custom renderer output
	if !strings.Contains(output, `<a href="https://example.com" title="Title" target="_blank" rel="noreferrer noopener">alt text</a>`) {
		t.Errorf("Custom link renderer not used. Output: %s", output)
	}
	if strings.Contains(output, `>alt text</a>alt text`) {
		t.Errorf("Link text duplicated, custom renderer not applied correctly. Output: %s", output)
	}
}
