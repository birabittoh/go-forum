package renderers

import (
	"bytes"
	"fmt"
	"html"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// CustomImageRenderer implements a custom image renderer
type CustomImageRenderer struct {
	goldmarkhtml.Config
}

// NewCustomImageRenderer creates a new custom renderer
func NewCustomImageRenderer(opts ...goldmarkhtml.Option) renderer.NodeRenderer {
	r := &CustomImageRenderer{
		Config: goldmarkhtml.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

// RegisterFuncs registers rendering functions
func (r *CustomImageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

// renderImage renders the image as a button-link
func (r *CustomImageRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*ast.Image)

	// Get the image URL
	destination := string(n.Destination)

	// Get the alt text
	altText := "Image"
	if n.Title != nil {
		altText = string(n.Title)
	} else {
		// If there's no title, use the text of child nodes
		var buf bytes.Buffer
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if textNode, ok := child.(*ast.Text); ok {
				buf.Write(textNode.Segment.Value(source))
			}
		}
		if buf.Len() > 0 {
			altText = buf.String()
		}
	}

	// Render as a centered button-link
	_, err := fmt.Fprintf(
		w,
		`<div class="text-center"><a href="%s" target="_blank" class="btn btn-secondary" rel="noreferrer noopener">🖼️ %s</a></div>`,
		html.EscapeString(destination),
		html.EscapeString(altText),
	)

	return ast.WalkContinue, err
}
