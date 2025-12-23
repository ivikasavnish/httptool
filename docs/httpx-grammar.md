# HTTPX Language Grammar

## Token Types

```
COMMENT         # ...
VAR             var
REQUEST         request
SCENARIO        scenario
LOAD            load
RUN             run
IF              if
ELSE            else
ASSERT          assert
EXTRACT         extract
RETRY           retry
CURL            curl
VUS             vus
RPS             rps
FOR             for
ITERATIONS      iterations
WITH            with
IN              in
STATUS          status
LATENCY         latency
BODY            body
MAX_ATTEMPTS    max_attempts
BACKOFF         backoff
BASE_DELAY      base_delay
THINK           think

IDENTIFIER      [a-zA-Z_][a-zA-Z0-9_]*
NUMBER          [0-9]+
STRING          "..." | '...'
DURATION        [0-9]+(ms|s|m|h)
VARIABLE_REF    ${identifier}

LBRACE          {
RBRACE          }
LPAREN          (
RPAREN          )
LBRACKET        [
RBRACKET        ]
ARROW           ->
EQUALS          =
DOUBLE_EQUALS   ==
NOT_EQUALS      !=
LT              <
GT              >
LTE             <=
GTE             >=
DOT             .
COMMA           ,
COLON           :
BACKSLASH       \
DOLLAR          $

NEWLINE         \n
EOF             end of file
```

## Grammar (EBNF)

```ebnf
program         ::= statement*

statement       ::= comment
                  | variable_declaration
                  | request_declaration
                  | scenario_declaration

comment         ::= '#' text NEWLINE

variable_declaration ::= 'var' IDENTIFIER '=' expression NEWLINE

request_declaration ::= 'request' IDENTIFIER '{' NEWLINE
                       request_body
                       '}'

request_body    ::= curl_command
                   (extract_block | assert_block | retry_block)*

curl_command    ::= 'curl' curl_args

curl_args       ::= url_or_flag*

extract_block   ::= 'extract' '{' NEWLINE
                   extraction_rule*
                   '}'

extraction_rule ::= IDENTIFIER '=' extraction_path NEWLINE

extraction_path ::= jsonpath        # $.field.subfield
                  | regex_pattern    # regex:pattern
                  | header_extract   # header:Header-Name
                  | cookie_extract   # cookie:cookie-name

assert_block    ::= 'assert' '{' NEWLINE
                   assertion*
                   '}'
                  | 'assert' assertion NEWLINE

assertion       ::= IDENTIFIER comparison_op expression NEWLINE
                  | IDENTIFIER 'in' '[' expression_list ']' NEWLINE

comparison_op   ::= '==' | '!=' | '<' | '>' | '<=' | '>='

retry_block     ::= 'retry' '{' NEWLINE
                   retry_config*
                   '}'

retry_config    ::= IDENTIFIER '=' expression NEWLINE

scenario_declaration ::= 'scenario' IDENTIFIER '{' NEWLINE
                        load_config
                        flow_statement*
                        '}'

load_config     ::= 'load' NUMBER 'vus' 'for' DURATION NEWLINE
                  | 'load' NUMBER 'rps' 'for' DURATION NEWLINE
                  | 'load' NUMBER 'iterations' 'with' NUMBER 'vus' NEWLINE
                  | 'load' '{' NEWLINE load_params* '}' NEWLINE

load_params     ::= IDENTIFIER '=' expression NEWLINE

flow_statement  ::= 'run' flow_expr

flow_expr       ::= sequential_flow
                  | nested_flow
                  | conditional_flow
                  | IDENTIFIER

sequential_flow ::= IDENTIFIER '->' flow_expr

nested_flow     ::= IDENTIFIER '{' NEWLINE
                   flow_statement*
                   '}'

conditional_flow ::= 'if' condition '{' NEWLINE
                    flow_statement*
                    '}'
                    ('else' '{' NEWLINE
                    flow_statement*
                    '}')?

condition       ::= expression comparison_op expression

expression      ::= STRING
                  | NUMBER
                  | DURATION
                  | VARIABLE_REF
                  | IDENTIFIER
                  | boolean

expression_list ::= expression (',' expression)*

boolean         ::= 'true' | 'false'
```

## AST Node Types

```go
type Node interface {
    Type() string
    Position() Position
}

type Position struct {
    Line   int
    Column int
}

// Top-level declarations
type Program struct {
    Statements []Statement
}

type Statement interface {
    Node
    statementNode()
}

type Comment struct {
    Text string
    Pos  Position
}

type VariableDeclaration struct {
    Name  string
    Value Expression
    Pos   Position
}

type RequestDeclaration struct {
    Name         string
    CurlCommand  *CurlCommand
    Extractions  []Extraction
    Assertions   []Assertion
    RetryConfig  *RetryConfig
    Pos          Position
}

type ScenarioDeclaration struct {
    Name      string
    LoadConfig *LoadConfig
    Flow      []FlowStatement
    Pos       Position
}

// Curl command
type CurlCommand struct {
    URL     string
    Method  string
    Headers map[string]string
    Body    string
    Cookies map[string]string
    Flags   []string
    Pos     Position
}

// Load configuration
type LoadConfig struct {
    VUs        int
    RPS        int
    Iterations int
    Duration   string
    Pos        Position
}

// Extraction
type Extraction struct {
    Variable string
    Path     string
    Type     string // "jsonpath", "regex", "header", "cookie"
    Pos      Position
}

// Assertion
type Assertion struct {
    Field    string
    Operator string
    Value    Expression
    Values   []Expression // for 'in' operator
    Pos      Position
}

// Retry configuration
type RetryConfig struct {
    MaxAttempts int
    Backoff     string
    BaseDelay   string
    Pos         Position
}

// Flow statements
type FlowStatement interface {
    Node
    flowNode()
}

type RunStatement struct {
    Request string
    Pos     Position
}

type SequentialFlow struct {
    Steps []string
    Pos   Position
}

type NestedFlow struct {
    Parent   string
    Children []FlowStatement
    Pos      Position
}

type ConditionalFlow struct {
    Condition  *Condition
    ThenBlock  []FlowStatement
    ElseBlock  []FlowStatement
    Pos        Position
}

// Expressions
type Expression interface {
    Node
    exprNode()
}

type StringLiteral struct {
    Value string
    Pos   Position
}

type NumberLiteral struct {
    Value int
    Pos   Position
}

type DurationLiteral struct {
    Value string
    Pos   Position
}

type VariableReference struct {
    Name string
    Pos  Position
}

type Identifier struct {
    Name string
    Pos  Position
}

type BooleanLiteral struct {
    Value bool
    Pos   Position
}

// Condition
type Condition struct {
    Left     Expression
    Operator string
    Right    Expression
    Pos      Position
}
```

## Lexer States

The lexer can be in different states:

1. **Normal** - Default state, recognizing keywords and identifiers
2. **InCurl** - Inside curl command, treat most tokens as raw arguments
3. **InString** - Inside string literal, handle escapes
4. **InComment** - Inside comment, ignore until newline

## Parser Strategy

1. **Lexer** produces stream of tokens
2. **Parser** consumes tokens to build AST
3. **AST** is validated for semantic correctness
4. **Compiler** transforms AST to IR (existing intermediate representation)
