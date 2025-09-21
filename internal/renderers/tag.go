package renderers

import (
	"fmt"
	"html"
	"regexp"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// MentionNode rappresenta una menzione @username nell'AST
type MentionNode struct {
	ast.BaseInline
	Username string
}

// Dump implementa ast.Node
func (n *MentionNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{
		"Username": n.Username,
	}, nil)
}

// Kind restituisce il tipo del nodo
func (n *MentionNode) Kind() ast.NodeKind {
	return KindMention
}

// KindMention è il tipo per i nodi menzione
var KindMention = ast.NewNodeKind("Mention")

// NewMentionNode crea un nuovo nodo menzione
func NewMentionNode(username string) *MentionNode {
	return &MentionNode{
		Username: username,
	}
}

// MentionParser parsa le menzioni @username
type MentionParser struct {
	usernameRegex *regexp.Regexp
}

// NewMentionParser crea un nuovo parser per le menzioni
func NewMentionParser() *MentionParser {
	pattern := `^[a-zA-Z0-9][a-zA-Z0-9_.-]{3,19}`
	return &MentionParser{
		usernameRegex: regexp.MustCompile(pattern),
	}
}

// Trigger restituisce i caratteri che attivano il parser
func (s *MentionParser) Trigger() []byte {
	return []byte{'@'}
}

// Parse parsa una menzione
func (s *MentionParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, segment := block.PeekLine()

	// Verifica che inizia con @
	if len(line) < 1 || line[0] != '@' {
		return nil
	}

	// Verifica che prima di @ ci sia spazio o inizio riga
	pos := segment.Start
	if pos > 0 {
		prevChar := block.Source()[pos-1]
		if !unicode.IsSpace(rune(prevChar)) && prevChar != '\n' && prevChar != '\r' {
			return nil
		}
	}

	// Estrae lo username dopo @
	rest := line[1:] // rimuove @
	match := s.usernameRegex.Find(rest)
	if len(match) == 0 {
		return nil
	}

	username := string(match)

	// Avanza il reader
	block.Advance(1 + len(match)) // +1 per @

	return NewMentionNode(username)
}

// MentionRenderer renderizza i nodi menzione
type MentionRenderer struct {
	goldmarkhtml.Config
}

// NewMentionRenderer crea un nuovo renderer per menzioni
func NewMentionRenderer(opts ...goldmarkhtml.Option) renderer.NodeRenderer {
	r := &MentionRenderer{
		Config: goldmarkhtml.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

// RegisterFuncs registra le funzioni di rendering
func (r *MentionRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindMention, r.renderMention)
}

// renderMention renderizza una menzione come link
func (r *MentionRenderer) renderMention(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*MentionNode)

	_, err := fmt.Fprintf(w,
		`<a href="/profile/%s">%s</a>`,
		html.EscapeString(n.Username),
		html.EscapeString(n.Username))

	return ast.WalkContinue, err
}

// MentionExtension è l'estensione per le menzioni
type MentionExtension struct{}

// Extend estende goldmark con il supporto per le menzioni
func (e *MentionExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithInlineParsers(
			util.Prioritized(NewMentionParser(), 100),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(NewMentionRenderer(), 100),
		),
	)
}
