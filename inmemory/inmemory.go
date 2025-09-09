package inmemory

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/letmevibethatforyou/searchx"
)

// Document represents a JSON document in the in-memory database.
type Document struct {
	// ID is the unique identifier for the document.
	ID string
	// Fields contains the document's data as key-value pairs.
	Fields map[string]interface{}
}

// Searcher implements the searchx.Searcher interface using an in-memory store.
type Searcher struct {
	mu        sync.RWMutex
	documents []Document
	idIndex   map[string]int // maps document ID to index in documents slice
}

// New creates a new in-memory searcher.
// The searcher is ready to use and is safe for concurrent operations.
func New() *Searcher {
	return &Searcher{
		documents: make([]Document, 0),
		idIndex:   make(map[string]int),
	}
}

// AddDocument adds a document to the in-memory store.
// If a document with the same ID already exists, it will be updated.
// This method is safe for concurrent use.
func (s *Searcher) AddDocument(doc Document) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if idx, exists := s.idIndex[doc.ID]; exists {
		// Update existing document
		s.documents[idx] = doc
	} else {
		// Add new document
		s.idIndex[doc.ID] = len(s.documents)
		s.documents = append(s.documents, doc)
	}
}

// AddJSON adds a JSON document to the in-memory store by parsing the provided JSON data.
// If a document with the same ID already exists, it will be updated.
// This method is safe for concurrent use.
func (s *Searcher) AddJSON(id string, jsonData []byte) error {
	var fields map[string]interface{}
	if err := json.Unmarshal(jsonData, &fields); err != nil {
		return errors.Wrap(err, "failed to unmarshal JSON")
	}

	s.AddDocument(Document{
		ID:     id,
		Fields: fields,
	})
	return nil
}

// RemoveDocument removes a document by ID from the in-memory store.
// Returns true if the document was found and removed, false if the document was not found.
// This method is safe for concurrent use.
func (s *Searcher) RemoveDocument(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, exists := s.idIndex[id]
	if !exists {
		return false
	}

	// Remove from slice
	s.documents = append(s.documents[:idx], s.documents[idx+1:]...)

	// Rebuild index
	delete(s.idIndex, id)
	for i := idx; i < len(s.documents); i++ {
		s.idIndex[s.documents[i].ID] = i
	}

	return true
}

// Clear removes all documents from the store.
// This method is safe for concurrent use.
func (s *Searcher) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.documents = make([]Document, 0)
	s.idIndex = make(map[string]int)
}

// Size returns the number of documents currently stored in the in-memory store.
// This method is safe for concurrent use.
func (s *Searcher) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.documents)
}

// Search implements the searchx.Searcher interface.
func (s *Searcher) Search(ctx context.Context, query string, opts ...searchx.SearchOption) (*searchx.Results, error) {
	startTime := time.Now()

	// Check context
	select {
	case <-ctx.Done():
		return nil, searchx.ErrCanceled
	default:
	}

	// Parse options
	cfg := &searchx.SearchConfig{}
	for _, opt := range opts {
		opt.Apply(cfg)
	}

	// Set defaults
	if cfg.Limit == 0 {
		cfg.Limit = 10
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Filter documents
	var matches []scoredDocument
	for _, doc := range s.documents {
		// Check context periodically
		select {
		case <-ctx.Done():
			return nil, searchx.ErrCanceled
		default:
		}

		// Apply filters
		if !s.matchesFilters(doc, cfg.Filters) {
			continue
		}

		// Apply query matching
		score := s.scoreDocument(doc, query)
		if score > 0 {
			matches = append(matches, scoredDocument{
				document: doc,
				score:    score,
			})
		}
	}

	// Sort matches
	s.sortMatches(matches, cfg.Sort)

	// Apply pagination
	total := int64(len(matches))
	start := cfg.Offset
	end := cfg.Offset + cfg.Limit
	if end > len(matches) {
		end = len(matches)
	}
	if start > len(matches) {
		start = len(matches)
	}

	// Build results
	results := &searchx.Results{
		Items: make([]searchx.Result, 0, end-start),
		Total: total,
		Query: query,
		Took:  time.Since(startTime).Milliseconds(),
	}

	// Convert matches to results
	maxScore := 0.0
	for i := start; i < end; i++ {
		match := matches[i]
		if match.score > maxScore {
			maxScore = match.score
		}
		results.Items = append(results.Items, searchx.Result{
			ID:     match.document.ID,
			Score:  match.score,
			Fields: match.document.Fields,
		})
	}
	results.MaxScore = maxScore

	// Set next offset for pagination
	if end < len(matches) {
		nextOffset := end
		results.NextOffset = &nextOffset
	}

	return results, nil
}

type scoredDocument struct {
	document Document
	score    float64
}

// scoreDocument calculates the relevance score for a document based on the query.
func (s *Searcher) scoreDocument(doc Document, query string) float64 {
	if query == "" {
		return 1.0 // All documents match empty query
	}

	query = strings.ToLower(query)
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return 1.0
	}

	score := 0.0
	matchedTerms := 0

	for _, term := range terms {
		termMatched := false
		for _, value := range doc.Fields {
			if s.valueContainsTerm(value, term) {
				termMatched = true
				score += 1.0
			}
		}
		if termMatched {
			matchedTerms++
		}
	}

	if matchedTerms == 0 {
		return 0
	}

	// Boost score if all terms matched
	if matchedTerms == len(terms) {
		score *= 1.5
	}

	return score
}

// valueContainsTerm checks if a value contains the search term.
func (s *Searcher) valueContainsTerm(value interface{}, term string) bool {
	switch v := value.(type) {
	case string:
		return strings.Contains(strings.ToLower(v), term)
	case []interface{}:
		for _, item := range v {
			if s.valueContainsTerm(item, term) {
				return true
			}
		}
	case map[string]interface{}:
		for _, item := range v {
			if s.valueContainsTerm(item, term) {
				return true
			}
		}
	default:
		// Convert to string and check
		str := fmt.Sprintf("%v", v)
		return strings.Contains(strings.ToLower(str), term)
	}
	return false
}

// sortMatches sorts the matched documents according to the sort configuration.
func (s *Searcher) sortMatches(matches []scoredDocument, sortFields []searchx.SortField) {
	if len(sortFields) == 0 {
		// Default: sort by score descending
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].score > matches[j].score
		})
		return
	}

	sort.Slice(matches, func(i, j int) bool {
		for _, sf := range sortFields {
			if sf.Field == "_score" {
				if matches[i].score != matches[j].score {
					if sf.Desc {
						return matches[i].score > matches[j].score
					}
					return matches[i].score < matches[j].score
				}
				continue
			}

			val1 := matches[i].document.Fields[sf.Field]
			val2 := matches[j].document.Fields[sf.Field]

			cmp := s.compareValues(val1, val2)
			if cmp != 0 {
				if sf.Desc {
					return cmp > 0
				}
				return cmp < 0
			}
		}
		return false
	})
}

// compareValues compares two values for sorting.
func (s *Searcher) compareValues(v1, v2 interface{}) int {
	// Handle nil values
	if v1 == nil && v2 == nil {
		return 0
	}
	if v1 == nil {
		return -1
	}
	if v2 == nil {
		return 1
	}

	// Try to compare as numbers
	if f1, ok1 := toFloat64(v1); ok1 {
		if f2, ok2 := toFloat64(v2); ok2 {
			if f1 < f2 {
				return -1
			} else if f1 > f2 {
				return 1
			}
			return 0
		}
	}

	// Compare as strings
	s1 := fmt.Sprintf("%v", v1)
	s2 := fmt.Sprintf("%v", v2)
	return strings.Compare(s1, s2)
}
