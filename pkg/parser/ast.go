package parser

// Node is the base interface for all AST nodes
type Node interface {
	TokenLiteral() string
	Position() Position
}

// Position represents a location in the source code
type Position struct {
	Line   int
	Column int
}

// Statement represents a statement node
type Statement interface {
	Node
	statementNode()
}

// Expression represents an expression node
type Expression interface {
	Node
	expressionNode()
}

// =========================================
// Top-Level Nodes
// =========================================

// Program represents the entire parsed program
type Program struct {
	Statements []Statement
	Pos        Position
}

func (p *Program) TokenLiteral() string { return "" }
func (p *Program) Position() Position   { return p.Pos }

// =========================================
// Statements
// =========================================

// Comment represents a comment
type Comment struct {
	Text string
	Pos  Position
}

func (c *Comment) TokenLiteral() string { return "#" }
func (c *Comment) Position() Position   { return c.Pos }
func (c *Comment) statementNode()       {}

// VariableDeclaration represents: var name = value
type VariableDeclaration struct {
	Name  string
	Value Expression
	Pos   Position
}

func (v *VariableDeclaration) TokenLiteral() string { return "var" }
func (v *VariableDeclaration) Position() Position   { return v.Pos }
func (v *VariableDeclaration) statementNode()       {}

// RequestDeclaration represents a request block
type RequestDeclaration struct {
	Name        string
	CurlCommand *CurlCommand
	Assertions  []*Assertion
	Extractions []*Extraction
	RetryConfig *RetryConfig
	Pos         Position
}

func (r *RequestDeclaration) TokenLiteral() string { return "request" }
func (r *RequestDeclaration) Position() Position   { return r.Pos }
func (r *RequestDeclaration) statementNode()       {}

// ScenarioDeclaration represents a scenario block
type ScenarioDeclaration struct {
	Name       string
	LoadConfig *LoadConfig
	Flow       []FlowStatement
	Pos        Position
}

func (s *ScenarioDeclaration) TokenLiteral() string { return "scenario" }
func (s *ScenarioDeclaration) Position() Position   { return s.Pos }
func (s *ScenarioDeclaration) statementNode()       {}

// =========================================
// Curl Command
// =========================================

// CurlCommand represents a curl command
type CurlCommand struct {
	URL         string              // The URL (may contain variable refs)
	URLParts    []Expression        // URL broken into parts (strings and var refs)
	Method      string              // HTTP method (GET, POST, etc)
	Headers     map[string]string   // Headers from -H flags
	Body        string              // Body from -d flag
	Cookies     map[string]string   // Cookies from -b flag
	RawArgs     []string            // All raw arguments
	Pos         Position
}

func (c *CurlCommand) TokenLiteral() string { return "curl" }
func (c *CurlCommand) Position() Position   { return c.Pos }

// =========================================
// Load Configuration
// =========================================

// LoadConfig represents load configuration
type LoadConfig struct {
	VUs        int
	RPS        int
	Iterations int
	Duration   string
	Pos        Position
}

func (l *LoadConfig) TokenLiteral() string { return "load" }
func (l *LoadConfig) Position() Position   { return l.Pos }

// =========================================
// Extraction
// =========================================

// Extraction represents variable extraction
type Extraction struct {
	Variable string
	Path     string
	Type     ExtractionType
	Pos      Position
}

type ExtractionType int

const (
	ExtractJSONPath ExtractionType = iota
	ExtractRegex
	ExtractHeader
	ExtractCookie
)

func (e *Extraction) TokenLiteral() string { return "extract" }
func (e *Extraction) Position() Position   { return e.Pos }

// =========================================
// Assertion
// =========================================

// Assertion represents an assertion
type Assertion struct {
	Field    string
	Operator string
	Value    Expression
	Values   []Expression // for 'in' operator
	Pos      Position
}

func (a *Assertion) TokenLiteral() string { return "assert" }
func (a *Assertion) Position() Position   { return a.Pos }

// =========================================
// Retry Configuration
// =========================================

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxAttempts int
	Backoff     string
	BaseDelay   string
	Pos         Position
}

func (r *RetryConfig) TokenLiteral() string { return "retry" }
func (r *RetryConfig) Position() Position   { return r.Pos }

// =========================================
// Flow Statements
// =========================================

// FlowStatement represents a flow control statement
type FlowStatement interface {
	Node
	flowNode()
}

// RunStatement represents: run request_name
type RunStatement struct {
	RequestName string
	Pos         Position
}

func (r *RunStatement) TokenLiteral() string { return "run" }
func (r *RunStatement) Position() Position   { return r.Pos }
func (r *RunStatement) flowNode()            {}

// SequentialFlow represents: run req1 -> req2 -> req3
type SequentialFlow struct {
	Steps []string
	Pos   Position
}

func (s *SequentialFlow) TokenLiteral() string { return "run" }
func (s *SequentialFlow) Position() Position   { return s.Pos }
func (s *SequentialFlow) flowNode()            {}

// NestedFlow represents: run parent { run child }
type NestedFlow struct {
	Parent   string
	Children []FlowStatement
	Pos      Position
}

func (n *NestedFlow) TokenLiteral() string { return "run" }
func (n *NestedFlow) Position() Position   { return n.Pos }
func (n *NestedFlow) flowNode()            {}

// ConditionalFlow represents: if condition { ... } else { ... }
type ConditionalFlow struct {
	Condition *Condition
	ThenBlock []FlowStatement
	ElseBlock []FlowStatement
	Pos       Position
}

func (c *ConditionalFlow) TokenLiteral() string { return "if" }
func (c *ConditionalFlow) Position() Position   { return c.Pos }
func (c *ConditionalFlow) flowNode()            {}

// =========================================
// Expressions
// =========================================

// StringLiteral represents a string literal
type StringLiteral struct {
	Value string
	Pos   Position
}

func (s *StringLiteral) TokenLiteral() string { return s.Value }
func (s *StringLiteral) Position() Position   { return s.Pos }
func (s *StringLiteral) expressionNode()      {}

// NumberLiteral represents a number literal
type NumberLiteral struct {
	Value int
	Pos   Position
}

func (n *NumberLiteral) TokenLiteral() string { return "" }
func (n *NumberLiteral) Position() Position   { return n.Pos }
func (n *NumberLiteral) expressionNode()      {}

// DurationLiteral represents a duration literal (5m, 30s, etc)
type DurationLiteral struct {
	Value string
	Pos   Position
}

func (d *DurationLiteral) TokenLiteral() string { return d.Value }
func (d *DurationLiteral) Position() Position   { return d.Pos }
func (d *DurationLiteral) expressionNode()      {}

// VariableReference represents a variable reference ${name}
type VariableReference struct {
	Name string
	Pos  Position
}

func (v *VariableReference) TokenLiteral() string { return v.Name }
func (v *VariableReference) Position() Position   { return v.Pos }
func (v *VariableReference) expressionNode()      {}

// Identifier represents an identifier
type Identifier struct {
	Name string
	Pos  Position
}

func (i *Identifier) TokenLiteral() string { return i.Name }
func (i *Identifier) Position() Position   { return i.Pos }
func (i *Identifier) expressionNode()      {}

// BooleanLiteral represents true/false
type BooleanLiteral struct {
	Value bool
	Pos   Position
}

func (b *BooleanLiteral) TokenLiteral() string {
	if b.Value {
		return "true"
	}
	return "false"
}
func (b *BooleanLiteral) Position() Position { return b.Pos }
func (b *BooleanLiteral) expressionNode()    {}

// =========================================
// Condition
// =========================================

// Condition represents a boolean condition
type Condition struct {
	Left     Expression
	Operator string
	Right    Expression
	Pos      Position
}

func (c *Condition) TokenLiteral() string { return c.Operator }
func (c *Condition) Position() Position   { return c.Pos }
