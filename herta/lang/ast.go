package lang

// AST for a Herta program. Positions are 1-based line:col.

type Program struct {
	Decls []Decl
}

type Decl interface{ declNode() }

type Runtime struct {
	Name      string
	Resources []ResourceDecl
	Line, Col int
}

type ResourceDecl struct {
	Name      string
	Capacity  int64
	Line, Col int
}

type Param struct {
	Name string
	Type string
}

type Op struct {
	Runtime   string
	Name      string
	Params    []Param
	Result    string
	Effect    string // "", "pure", "idempotent", "nonidempotent"
	Admission string // "", "wait", "reject"
	Resources []OpResource
	Timeout   string // duration literal text, "" if unset
	Key       string // serialize key param name, "" if unset
	Retry     *RetryDecl
	Line, Col int
}

type OpResource struct {
	Name      string
	Units     int64
	Line, Col int
}

type RetryDecl struct {
	Max       int64
	On        []OutcomeRef
	Line, Col int
}

type OutcomeRef struct {
	Name      string // transient, uncertain, permanent, throttled, success
	Line, Col int
}

type Handler struct {
	Runtime   string
	Name      string
	Params    []Param
	Results   []string // result types, last is expected "error" — checked
	Body      []Stmt
	Line, Col int
}

type Stmt interface{ stmtNode() }

// FailStmt is `fail <outcome>, <expr>` — the only contract-relevant stmt.
type FailStmt struct {
	Outcome   string
	Arg       string // raw expression text (kept opaque in v0.1)
	Line, Col int
}

// RawStmt is any other statement, preserved as source text for codegen.
// Fails records `fail` uses buried in inline blocks so the checker sees
// them; they print as part of Text, not as separate statements.
type RawStmt struct {
	Text      string
	Fails     []OutcomeRef
	Line, Col int
}

func (*Runtime) declNode()  {}
func (*Op) declNode()       {}
func (*Handler) declNode()  {}
func (*FailStmt) stmtNode() {}
func (*RawStmt) stmtNode()  {}
