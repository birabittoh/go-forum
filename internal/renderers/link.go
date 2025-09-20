package renderers

import (
	"fmt"
	"html"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// CustomLinkRenderer implements a custom link renderer
type CustomLinkRenderer struct {
	goldmarkhtml.Config
}

// NewCustomLinkRenderer creates a new custom renderer
func NewCustomLinkRenderer(opts ...goldmarkhtml.Option) renderer.NodeRenderer {
	r := &CustomLinkRenderer{
		Config: goldmarkhtml.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

// RegisterFuncs registers rendering functions
func (r *CustomLinkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.renderLink)
}

// renderLink renders the link with target and rel attributes
func (r *CustomLinkRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	destination := string(n.Destination)
	titleText := ""
	if n.Title != nil {
		titleText = string(n.Title)
	}

	if entering {
		_, err := fmt.Fprintf(
			w,
			`<a href="%s" title="%s" target="_blank" rel="noreferrer noopener">`,
			html.EscapeString(destination),
			html.EscapeString(titleText),
		)
		return ast.WalkContinue, err
	} else {
		_, err := fmt.Fprintf(w, "</a>")
		return ast.WalkContinue, err
	}
}
