package inmemory

import (
	"fmt"

	"github.com/letmevibethatforyou/searchx"
)

// matchesFilters checks if a document matches all the filter expressions.
func (s *Searcher) matchesFilters(doc Document, filters []searchx.Expression) bool {
	for _, filter := range filters {
		if !s.evaluateExpression(doc, filter) {
			return false
		}
	}
	return true
}

// evaluateExpression evaluates a single expression against a document.
func (s *Searcher) evaluateExpression(doc Document, expr searchx.Expression) bool {
	switch e := expr.(type) {
	case searchx.AndExpr:
		return s.evaluateAnd(doc, e)
	case searchx.OrExpr:
		return s.evaluateOr(doc, e)
	case searchx.NotExpr:
		return s.evaluateNot(doc, e)
	case searchx.EqExpr:
		return s.evaluateEq(doc, e)
	case searchx.NeExpr:
		return s.evaluateNe(doc, e)
	case searchx.GtExpr:
		return s.evaluateGt(doc, e)
	case searchx.GteExpr:
		return s.evaluateGte(doc, e)
	case searchx.LtExpr:
		return s.evaluateLt(doc, e)
	case searchx.LteExpr:
		return s.evaluateLte(doc, e)
	case searchx.RangeExpr:
		return s.evaluateRange(doc, e)
	case searchx.ExistsExpr:
		return s.evaluateExists(doc, e)
	default:
		// Unknown expression type, return true to not filter out
		return true
	}
}

// evaluateAnd evaluates an AND expression.
func (s *Searcher) evaluateAnd(doc Document, expr searchx.AndExpr) bool {
	for _, e := range expr.Exprs {
		if !s.evaluateExpression(doc, e) {
			return false
		}
	}
	return true
}

// evaluateOr evaluates an OR expression.
func (s *Searcher) evaluateOr(doc Document, expr searchx.OrExpr) bool {
	for _, e := range expr.Exprs {
		if s.evaluateExpression(doc, e) {
			return true
		}
	}
	return false
}

// evaluateNot evaluates a NOT expression.
func (s *Searcher) evaluateNot(doc Document, expr searchx.NotExpr) bool {
	return !s.evaluateExpression(doc, expr.Inner)
}

// evaluateEq evaluates an equality expression.
func (s *Searcher) evaluateEq(doc Document, expr searchx.EqExpr) bool {
	docValue, exists := doc.Fields[expr.Field]
	if !exists {
		return expr.Value == nil
	}

	return s.compareEqual(docValue, expr.Value)
}

// evaluateNe evaluates a not-equal expression.
func (s *Searcher) evaluateNe(doc Document, expr searchx.NeExpr) bool {
	docValue, exists := doc.Fields[expr.Field]
	if !exists {
		return expr.Value != nil
	}

	return !s.compareEqual(docValue, expr.Value)
}

// evaluateGt evaluates a greater-than expression.
func (s *Searcher) evaluateGt(doc Document, expr searchx.GtExpr) bool {
	docValue, exists := doc.Fields[expr.Field]
	if !exists {
		return false
	}

	return s.compareValues(docValue, expr.Value) > 0
}

// evaluateGte evaluates a greater-than-or-equal expression.
func (s *Searcher) evaluateGte(doc Document, expr searchx.GteExpr) bool {
	docValue, exists := doc.Fields[expr.Field]
	if !exists {
		return false
	}

	return s.compareValues(docValue, expr.Value) >= 0
}

// evaluateLt evaluates a less-than expression.
func (s *Searcher) evaluateLt(doc Document, expr searchx.LtExpr) bool {
	docValue, exists := doc.Fields[expr.Field]
	if !exists {
		return false
	}

	return s.compareValues(docValue, expr.Value) < 0
}

// evaluateLte evaluates a less-than-or-equal expression.
func (s *Searcher) evaluateLte(doc Document, expr searchx.LteExpr) bool {
	docValue, exists := doc.Fields[expr.Field]
	if !exists {
		return false
	}

	return s.compareValues(docValue, expr.Value) <= 0
}

// evaluateRange evaluates a range expression.
func (s *Searcher) evaluateRange(doc Document, expr searchx.RangeExpr) bool {
	docValue, exists := doc.Fields[expr.Field]
	if !exists {
		return false
	}

	if expr.Min != nil && s.compareValues(docValue, expr.Min) < 0 {
		return false
	}

	if expr.Max != nil && s.compareValues(docValue, expr.Max) > 0 {
		return false
	}

	return true
}

// evaluateExists evaluates an exists expression.
func (s *Searcher) evaluateExists(doc Document, expr searchx.ExistsExpr) bool {
	_, exists := doc.Fields[expr.Field]
	return exists
}

// compareEqual checks if two values are equal.
func (s *Searcher) compareEqual(v1, v2 interface{}) bool {
	// Handle nil cases
	if v1 == nil || v2 == nil {
		return v1 == v2
	}

	// Try numeric comparison
	if f1, ok1 := toFloat64(v1); ok1 {
		if f2, ok2 := toFloat64(v2); ok2 {
			return f1 == f2
		}
	}

	// Fall back to string comparison
	return fmt.Sprintf("%v", v1) == fmt.Sprintf("%v", v2)
}

// toFloat64 attempts to convert a value to float64.
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int8:
		return float64(val), true
	case int16:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint8:
		return float64(val), true
	case uint16:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	default:
		return 0, false
	}
}
