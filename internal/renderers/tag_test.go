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

func TestCustomTagRenderer(t *testing.T) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.CJK,
			&MentionExtension{},
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

	cases := []struct {
		name    string
		input   string
		want    string
		notWant string
	}{
		{
			name:  "admin42",
			input: "Hey @admin42, how are you?",
			want:  `<a href="/profile/admin42">admin42</a>`,
		},
		{
			name:    "my",
			input:   "@my mom is doing good.",
			notWant: `<a href="/profile/my">my</a>`,
		},
		{
			name:    "email",
			input:   "Email: test@example.com (should not be a link)",
			notWant: `/profile/example.com`,
		},
		{
			name:  "user_name",
			input: "@user_name is a valid user.",
			want:  `<a href="/profile/user_name">user_name</a>`,
		},
		{
			name:  "another-user",
			input: "@another-user is a valid user.",
			want:  `<a href="/profile/another-user">another-user</a>`,
		},
		{
			name:  "123valid",
			input: "@123valid is also a valid user.",
			want:  `<a href="/profile/123valid">123valid</a>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf strings.Builder
			if err := md.Convert([]byte(tc.input), &buf); err != nil {
				t.Fatalf("Markdown conversion failed: %v", err)
			}
			output := buf.String()
			if tc.want != "" && !strings.Contains(output, tc.want) {
				t.Errorf("Mention not rendered correctly for %s. Output: %s", tc.name, output)
			}
			if tc.notWant != "" && strings.Contains(output, tc.notWant) {
				t.Errorf("Mention incorrectly rendered for %s. Output: %s", tc.name, output)
			}
		})
	}
}
