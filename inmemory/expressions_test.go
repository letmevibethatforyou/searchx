package inmemory

import (
	"testing"

	"github.com/letmevibethatforyou/searchx"
)

func TestExpressionEvaluation(t *testing.T) {
	// Setup test documents
	doc1 := Document{
		ID: "1",
		Fields: map[string]interface{}{
			"name":     "John Doe",
			"age":      32,
			"score":    85.5,
			"active":   true,
			"tags":     []string{"developer", "golang"},
			"location": "New York",
		},
	}

	doc2 := Document{
		ID: "2",
		Fields: map[string]interface{}{
			"name":   "Jane Smith",
			"age":    25,
			"score":  92.0,
			"active": false,
		},
	}

	doc3 := Document{
		ID: "3",
		Fields: map[string]interface{}{
			"name":   "Bob Johnson",
			"age":    35,
			"score":  78.5,
			"active": true,
			"city":   "Boston",
		},
	}

	searcher := New()

	tests := map[string]struct {
		doc      Document
		expr     searchx.Expression
		expected bool
	}{
		// Equality tests
		"eq_string_match": {
			doc:      doc1,
			expr:     searchx.Eq("name", "John Doe"),
			expected: true,
		},
		"eq_string_no_match": {
			doc:      doc1,
			expr:     searchx.Eq("name", "Jane Smith"),
			expected: false,
		},
		"eq_number_match": {
			doc:      doc1,
			expr:     searchx.Eq("age", 32),
			expected: true,
		},
		"eq_float_match": {
			doc:      doc1,
			expr:     searchx.Eq("score", 85.5),
			expected: true,
		},
		"eq_bool_match": {
			doc:      doc1,
			expr:     searchx.Eq("active", true),
			expected: true,
		},
		"eq_field_not_exists": {
			doc:      doc2,
			expr:     searchx.Eq("location", "New York"),
			expected: false,
		},
		"eq_nil_value": {
			doc:      doc2,
			expr:     searchx.Eq("location", nil),
			expected: true,
		},

		// Not equal tests
		"ne_string_different": {
			doc:      doc1,
			expr:     searchx.Ne("name", "Jane Smith"),
			expected: true,
		},
		"ne_string_same": {
			doc:      doc1,
			expr:     searchx.Ne("name", "John Doe"),
			expected: false,
		},
		"ne_number_different": {
			doc:      doc1,
			expr:     searchx.Ne("age", 25),
			expected: true,
		},
		"ne_field_not_exists": {
			doc:      doc2,
			expr:     searchx.Ne("location", "New York"),
			expected: true,
		},

		// Greater than tests
		"gt_number_greater": {
			doc:      doc1,
			expr:     searchx.Gt("age", 25),
			expected: true,
		},
		"gt_number_equal": {
			doc:      doc1,
			expr:     searchx.Gt("age", 32),
			expected: false,
		},
		"gt_number_less": {
			doc:      doc1,
			expr:     searchx.Gt("age", 35),
			expected: false,
		},
		"gt_float_greater": {
			doc:      doc1,
			expr:     searchx.Gt("score", 80.0),
			expected: true,
		},
		"gt_field_not_exists": {
			doc:      doc2,
			expr:     searchx.Gt("location", 0),
			expected: false,
		},

		// Greater than or equal tests
		"gte_number_greater": {
			doc:      doc1,
			expr:     searchx.Gte("age", 25),
			expected: true,
		},
		"gte_number_equal": {
			doc:      doc1,
			expr:     searchx.Gte("age", 30),
			expected: true,
		},
		"gte_number_less": {
			doc:      doc1,
			expr:     searchx.Gte("age", 35),
			expected: false,
		},

		// Less than tests
		"lt_number_less": {
			doc:      doc1,
			expr:     searchx.Lt("age", 35),
			expected: true,
		},
		"lt_number_equal": {
			doc:      doc1,
			expr:     searchx.Lt("age", 30),
			expected: false,
		},
		"lt_number_greater": {
			doc:      doc1,
			expr:     searchx.Lt("age", 25),
			expected: false,
		},
		"lt_float_less": {
			doc:      doc1,
			expr:     searchx.Lt("score", 90.0),
			expected: true,
		},

		// Less than or equal tests
		"lte_number_less": {
			doc:      doc1,
			expr:     searchx.Lte("age", 35),
			expected: true,
		},
		"lte_number_equal": {
			doc:      doc1,
			expr:     searchx.Lte("age", 32),
			expected: true,
		},
		"lte_number_greater": {
			doc:      doc1,
			expr:     searchx.Lte("age", 25),
			expected: false,
		},

		// Range tests
		"range_within": {
			doc:      doc1,
			expr:     searchx.Range("age", 25, 35),
			expected: true,
		},
		"range_at_min": {
			doc:      doc1,
			expr:     searchx.Range("age", 30, 35),
			expected: true,
		},
		"range_at_max": {
			doc:      doc1,
			expr:     searchx.Range("age", 25, 32),
			expected: true,
		},
		"range_below": {
			doc:      doc1,
			expr:     searchx.Range("age", 35, 40),
			expected: false,
		},
		"range_above": {
			doc:      doc1,
			expr:     searchx.Range("age", 20, 25),
			expected: false,
		},
		"range_float_within": {
			doc:      doc1,
			expr:     searchx.Range("score", 80.0, 90.0),
			expected: true,
		},
		"range_nil_min": {
			doc:      doc1,
			expr:     searchx.Range("age", nil, 35),
			expected: true,
		},
		"range_nil_max": {
			doc:      doc1,
			expr:     searchx.Range("age", 25, nil),
			expected: true,
		},
		"range_field_not_exists": {
			doc:      doc2,
			expr:     searchx.Range("location", 0, 100),
			expected: false,
		},

		// Exists tests
		"exists_field_present": {
			doc:      doc1,
			expr:     searchx.Exists("name"),
			expected: true,
		},
		"exists_field_absent": {
			doc:      doc1,
			expr:     searchx.Exists("missing_field"),
			expected: false,
		},
		"exists_field_in_one_doc": {
			doc:      doc1,
			expr:     searchx.Exists("location"),
			expected: true,
		},
		"exists_field_not_in_other_doc": {
			doc:      doc2,
			expr:     searchx.Exists("location"),
			expected: false,
		},

		// AND expressions
		"and_all_true": {
			doc: doc1,
			expr: searchx.And(
				searchx.Eq("active", true),
				searchx.Gt("age", 30),
				searchx.Lt("score", 90),
			),
			expected: true,
		},
		"and_one_false": {
			doc: doc1,
			expr: searchx.And(
				searchx.Eq("active", true),
				searchx.Gt("age", 35), // false
				searchx.Lt("score", 90),
			),
			expected: false,
		},
		"and_all_false": {
			doc: doc1,
			expr: searchx.And(
				searchx.Eq("active", false),
				searchx.Lt("age", 25),
				searchx.Gt("score", 90),
			),
			expected: false,
		},
		"and_empty": {
			doc:      doc1,
			expr:     searchx.And(),
			expected: true,
		},

		// OR expressions
		"or_all_true": {
			doc: doc1,
			expr: searchx.Or(
				searchx.Eq("active", true),
				searchx.Gt("age", 25),
				searchx.Lt("score", 90),
			),
			expected: true,
		},
		"or_one_true": {
			doc: doc1,
			expr: searchx.Or(
				searchx.Eq("active", false),
				searchx.Gt("age", 35),
				searchx.Lt("score", 90), // true
			),
			expected: true,
		},
		"or_all_false": {
			doc: doc1,
			expr: searchx.Or(
				searchx.Eq("active", false),
				searchx.Lt("age", 25),
				searchx.Gt("score", 90),
			),
			expected: false,
		},
		"or_empty": {
			doc:      doc1,
			expr:     searchx.Or(),
			expected: false,
		},

		// NOT expressions
		"not_true": {
			doc:      doc1,
			expr:     searchx.Not(searchx.Eq("active", false)),
			expected: true,
		},
		"not_false": {
			doc:      doc1,
			expr:     searchx.Not(searchx.Eq("active", true)),
			expected: false,
		},
		"not_complex": {
			doc: doc1,
			expr: searchx.Not(
				searchx.And(
					searchx.Eq("active", true),
					searchx.Gt("age", 35),
				),
			),
			expected: true,
		},

		// Nested complex expressions
		"complex_nested_1": {
			doc: doc1,
			expr: searchx.And(
				searchx.Or(
					searchx.Eq("name", "John Doe"),
					searchx.Eq("name", "Jane Smith"),
				),
				searchx.Gte("age", 30),
				searchx.Not(searchx.Eq("active", false)),
			),
			expected: true,
		},
		"complex_nested_2": {
			doc: doc1,
			expr: searchx.Or(
				searchx.And(
					searchx.Eq("active", true),
					searchx.Range("age", 25, 35),
				),
				searchx.And(
					searchx.Eq("active", false),
					searchx.Gt("score", 90),
				),
			),
			expected: true,
		},
		"complex_nested_3": {
			doc: doc2,
			expr: searchx.Not(
				searchx.Or(
					searchx.Exists("location"),
					searchx.And(
						searchx.Eq("active", true),
						searchx.Lt("age", 30),
					),
				),
			),
			expected: true,
		},

		// String comparison edge cases
		"eq_string_case_sensitive": {
			doc:      doc1,
			expr:     searchx.Eq("name", "john doe"),
			expected: false,
		},
		"eq_string_with_spaces": {
			doc:      doc1,
			expr:     searchx.Eq("location", "New York"),
			expected: true,
		},

		// Numeric type coercion tests
		"eq_int_float_coercion": {
			doc:      doc1,
			expr:     searchx.Eq("age", 32.0),
			expected: true,
		},
		"gt_float_int_coercion": {
			doc:      doc1,
			expr:     searchx.Gt("score", 85),
			expected: true,
		},

		// Different document tests
		"doc2_age_check": {
			doc:      doc2,
			expr:     searchx.Lt("age", 30),
			expected: true,
		},
		"doc2_active_check": {
			doc:      doc2,
			expr:     searchx.Eq("active", false),
			expected: true,
		},
		"doc3_city_exists": {
			doc:      doc3,
			expr:     searchx.Exists("city"),
			expected: true,
		},
		"doc3_location_not_exists": {
			doc:      doc3,
			expr:     searchx.Exists("location"),
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := searcher.evaluateExpression(tc.doc, tc.expr)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for expression on doc %s", tc.expected, result, tc.doc.ID)
			}
		})
	}
}

func TestCompareValues(t *testing.T) {
	searcher := New()

	tests := map[string]struct {
		v1       interface{}
		v2       interface{}
		expected int
	}{
		"equal_ints": {
			v1:       5,
			v2:       5,
			expected: 0,
		},
		"less_ints": {
			v1:       3,
			v2:       5,
			expected: -1,
		},
		"greater_ints": {
			v1:       7,
			v2:       5,
			expected: 1,
		},
		"equal_floats": {
			v1:       5.5,
			v2:       5.5,
			expected: 0,
		},
		"less_floats": {
			v1:       3.5,
			v2:       5.5,
			expected: -1,
		},
		"greater_floats": {
			v1:       7.5,
			v2:       5.5,
			expected: 1,
		},
		"int_float_equal": {
			v1:       5,
			v2:       5.0,
			expected: 0,
		},
		"int_float_less": {
			v1:       5,
			v2:       5.1,
			expected: -1,
		},
		"equal_strings": {
			v1:       "apple",
			v2:       "apple",
			expected: 0,
		},
		"less_strings": {
			v1:       "apple",
			v2:       "banana",
			expected: -1,
		},
		"greater_strings": {
			v1:       "banana",
			v2:       "apple",
			expected: 1,
		},
		"both_nil": {
			v1:       nil,
			v2:       nil,
			expected: 0,
		},
		"first_nil": {
			v1:       nil,
			v2:       "value",
			expected: -1,
		},
		"second_nil": {
			v1:       "value",
			v2:       nil,
			expected: 1,
		},
		"bool_vs_string": {
			v1:       true,
			v2:       "true",
			expected: 0,
		},
		"uint_types": {
			v1:       uint(10),
			v2:       uint32(10),
			expected: 0,
		},
		"int8_type": {
			v1:       int8(10),
			v2:       10,
			expected: 0,
		},
		"int16_type": {
			v1:       int16(10),
			v2:       10,
			expected: 0,
		},
		"int32_type": {
			v1:       int32(10),
			v2:       10,
			expected: 0,
		},
		"int64_type": {
			v1:       int64(10),
			v2:       10,
			expected: 0,
		},
		"uint8_type": {
			v1:       uint8(10),
			v2:       10,
			expected: 0,
		},
		"uint16_type": {
			v1:       uint16(10),
			v2:       10,
			expected: 0,
		},
		"uint32_type": {
			v1:       uint32(10),
			v2:       10,
			expected: 0,
		},
		"uint64_type": {
			v1:       uint64(10),
			v2:       10,
			expected: 0,
		},
		"float32_type": {
			v1:       float32(10.5),
			v2:       10.5,
			expected: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := searcher.compareValues(tc.v1, tc.v2)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d for comparing %v and %v", tc.expected, result, tc.v1, tc.v2)
			}
		})
	}
}

func TestCompareEqual(t *testing.T) {
	searcher := New()

	tests := map[string]struct {
		v1       interface{}
		v2       interface{}
		expected bool
	}{
		"equal_ints": {
			v1:       5,
			v2:       5,
			expected: true,
		},
		"unequal_ints": {
			v1:       5,
			v2:       3,
			expected: false,
		},
		"equal_floats": {
			v1:       5.5,
			v2:       5.5,
			expected: true,
		},
		"int_float_equal": {
			v1:       5,
			v2:       5.0,
			expected: true,
		},
		"equal_strings": {
			v1:       "test",
			v2:       "test",
			expected: true,
		},
		"unequal_strings": {
			v1:       "test",
			v2:       "Test",
			expected: false,
		},
		"both_nil": {
			v1:       nil,
			v2:       nil,
			expected: true,
		},
		"one_nil": {
			v1:       nil,
			v2:       "value",
			expected: false,
		},
		"bool_true": {
			v1:       true,
			v2:       true,
			expected: true,
		},
		"bool_false": {
			v1:       false,
			v2:       false,
			expected: true,
		},
		"bool_different": {
			v1:       true,
			v2:       false,
			expected: false,
		},
		"string_number_comparison": {
			v1:       "5",
			v2:       5,
			expected: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := searcher.compareEqual(tc.v1, tc.v2)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for comparing %v and %v", tc.expected, result, tc.v1, tc.v2)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := map[string]struct {
		value    interface{}
		expected float64
		ok       bool
	}{
		"float64": {
			value:    float64(10.5),
			expected: 10.5,
			ok:       true,
		},
		"float32": {
			value:    float32(10.5),
			expected: 10.5,
			ok:       true,
		},
		"int": {
			value:    10,
			expected: 10.0,
			ok:       true,
		},
		"int8": {
			value:    int8(10),
			expected: 10.0,
			ok:       true,
		},
		"int16": {
			value:    int16(10),
			expected: 10.0,
			ok:       true,
		},
		"int32": {
			value:    int32(10),
			expected: 10.0,
			ok:       true,
		},
		"int64": {
			value:    int64(10),
			expected: 10.0,
			ok:       true,
		},
		"uint": {
			value:    uint(10),
			expected: 10.0,
			ok:       true,
		},
		"uint8": {
			value:    uint8(10),
			expected: 10.0,
			ok:       true,
		},
		"uint16": {
			value:    uint16(10),
			expected: 10.0,
			ok:       true,
		},
		"uint32": {
			value:    uint32(10),
			expected: 10.0,
			ok:       true,
		},
		"uint64": {
			value:    uint64(10),
			expected: 10.0,
			ok:       true,
		},
		"string": {
			value:    "10.5",
			expected: 0,
			ok:       false,
		},
		"bool": {
			value:    true,
			expected: 0,
			ok:       false,
		},
		"nil": {
			value:    nil,
			expected: 0,
			ok:       false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, ok := toFloat64(tc.value)
			if ok != tc.ok {
				t.Errorf("Expected ok=%v, got %v for value %v", tc.ok, ok, tc.value)
			}
			if ok && result != tc.expected {
				t.Errorf("Expected %f, got %f for value %v", tc.expected, result, tc.value)
			}
		})
	}
}
