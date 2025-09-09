package searchx

// Expression represents a composable filter expression.
// All Expressions are SearchOptions, but not all SearchOptions are Expressions.
type Expression interface {
	SearchOption
	// expr is a marker method to distinguish expressions from other options.
	expr()
}

// baseExpr provides the expr marker method for all expression types.
type baseExpr struct{}

func (baseExpr) expr() {}

// AndExpr represents an AND combination of expressions.
type AndExpr struct {
	baseExpr
	// Exprs contains the expressions to combine with AND logic.
	Exprs []Expression
}

// Apply implements the SearchOption interface for AndExpr.
func (a AndExpr) Apply(cfg *SearchConfig) {
	cfg.Filters = append(cfg.Filters, a)
}

// And creates an AND expression combining multiple expressions.
func And(exprs ...Expression) Expression {
	return AndExpr{Exprs: exprs}
}

// OrExpr represents an OR combination of expressions.
type OrExpr struct {
	baseExpr
	// Exprs contains the expressions to combine with OR logic.
	Exprs []Expression
}

// Apply implements the SearchOption interface for OrExpr.
func (o OrExpr) Apply(cfg *SearchConfig) {
	cfg.Filters = append(cfg.Filters, o)
}

// Or creates an OR expression combining multiple expressions.
func Or(exprs ...Expression) Expression {
	return OrExpr{Exprs: exprs}
}

// NotExpr represents a NOT negation of an expression.
type NotExpr struct {
	baseExpr
	// Inner is the expression to negate.
	Inner Expression
}

// Apply implements the SearchOption interface for NotExpr.
func (n NotExpr) Apply(cfg *SearchConfig) {
	cfg.Filters = append(cfg.Filters, n)
}

// Not creates a NOT expression negating the given expression.
func Not(expr Expression) Expression {
	return NotExpr{Inner: expr}
}

// EqExpr represents an equality comparison expression.
type EqExpr struct {
	baseExpr
	// Field is the name of the field to compare.
	Field string
	// Value is the value to compare against.
	Value interface{}
}

// Apply implements the SearchOption interface for EqExpr.
func (e EqExpr) Apply(cfg *SearchConfig) {
	cfg.Filters = append(cfg.Filters, e)
}

// Eq creates an equality comparison expression.
func Eq(field string, value interface{}) Expression {
	return EqExpr{Field: field, Value: value}
}

// NeExpr represents a not-equal comparison expression.
type NeExpr struct {
	baseExpr
	// Field is the name of the field to compare.
	Field string
	// Value is the value to compare against.
	Value interface{}
}

// Apply implements the SearchOption interface for NeExpr.
func (n NeExpr) Apply(cfg *SearchConfig) {
	cfg.Filters = append(cfg.Filters, n)
}

// Ne creates a not-equal comparison expression.
func Ne(field string, value interface{}) Expression {
	return NeExpr{Field: field, Value: value}
}

// GtExpr represents a greater-than comparison expression.
type GtExpr struct {
	baseExpr
	// Field is the name of the field to compare.
	Field string
	// Value is the value to compare against.
	Value interface{}
}

// Apply implements the SearchOption interface for GtExpr.
func (g GtExpr) Apply(cfg *SearchConfig) {
	cfg.Filters = append(cfg.Filters, g)
}

// Gt creates a greater-than comparison expression.
func Gt(field string, value interface{}) Expression {
	return GtExpr{Field: field, Value: value}
}

// GteExpr represents a greater-than-or-equal comparison expression.
type GteExpr struct {
	baseExpr
	// Field is the name of the field to compare.
	Field string
	// Value is the value to compare against.
	Value interface{}
}

// Apply implements the SearchOption interface for GteExpr.
func (g GteExpr) Apply(cfg *SearchConfig) {
	cfg.Filters = append(cfg.Filters, g)
}

// Gte creates a greater-than-or-equal comparison expression.
func Gte(field string, value interface{}) Expression {
	return GteExpr{Field: field, Value: value}
}

// LtExpr represents a less-than comparison expression.
type LtExpr struct {
	baseExpr
	// Field is the name of the field to compare.
	Field string
	// Value is the value to compare against.
	Value interface{}
}

// Apply implements the SearchOption interface for LtExpr.
func (l LtExpr) Apply(cfg *SearchConfig) {
	cfg.Filters = append(cfg.Filters, l)
}

// Lt creates a less-than comparison expression.
func Lt(field string, value interface{}) Expression {
	return LtExpr{Field: field, Value: value}
}

// LteExpr represents a less-than-or-equal comparison expression.
type LteExpr struct {
	baseExpr
	// Field is the name of the field to compare.
	Field string
	// Value is the value to compare against.
	Value interface{}
}

// Apply implements the SearchOption interface for LteExpr.
func (l LteExpr) Apply(cfg *SearchConfig) {
	cfg.Filters = append(cfg.Filters, l)
}

// Lte creates a less-than-or-equal comparison expression.
func Lte(field string, value interface{}) Expression {
	return LteExpr{Field: field, Value: value}
}

// RangeExpr represents a range comparison expression.
type RangeExpr struct {
	baseExpr
	// Field is the name of the field to compare.
	Field string
	// Min is the minimum value of the range (inclusive). Can be nil for no lower bound.
	Min interface{}
	// Max is the maximum value of the range (inclusive). Can be nil for no upper bound.
	Max interface{}
}

// Apply implements the SearchOption interface for RangeExpr.
func (r RangeExpr) Apply(cfg *SearchConfig) {
	cfg.Filters = append(cfg.Filters, r)
}

// Range creates a range comparison expression.
func Range(field string, min, max interface{}) Expression {
	return RangeExpr{Field: field, Min: min, Max: max}
}

// ExistsExpr represents a field existence check expression.
type ExistsExpr struct {
	baseExpr
	// Field is the name of the field to check for existence.
	Field string
}

// Apply implements the SearchOption interface for ExistsExpr.
func (e ExistsExpr) Apply(cfg *SearchConfig) {
	cfg.Filters = append(cfg.Filters, e)
}

// Exists creates a field existence check expression.
func Exists(field string) Expression {
	return ExistsExpr{Field: field}
}
