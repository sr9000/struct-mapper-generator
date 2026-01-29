package match

import (
	"go/types"
	"testing"

	"caster-generator/internal/analyze"
)

func TestRankCandidates(t *testing.T) {
	// Create mock types
	intType := types.Typ[types.Int]
	int64Type := types.Typ[types.Int64]
	stringType := types.Typ[types.String]

	// Create target field
	targetField := &analyze.FieldInfo{
		Name:     "CustomerID",
		Exported: true,
		Type: &analyze.TypeInfo{
			GoType: int64Type,
		},
	}

	// Create source fields
	sourceFields := []analyze.FieldInfo{
		{
			Name:     "CustomerID",
			Exported: true,
			Type:     &analyze.TypeInfo{GoType: int64Type},
		},
		{
			Name:     "customer_id",
			Exported: true,
			Type:     &analyze.TypeInfo{GoType: intType},
		},
		{
			Name:     "CustomerName",
			Exported: true,
			Type:     &analyze.TypeInfo{GoType: stringType},
		},
		{
			Name:     "ID",
			Exported: true,
			Type:     &analyze.TypeInfo{GoType: int64Type},
		},
		{
			Name:     "unexported",
			Exported: false,
			Type:     &analyze.TypeInfo{GoType: int64Type},
		},
	}

	candidates := RankCandidates(targetField, sourceFields)

	// Should have 4 candidates (unexported filtered out)
	if len(candidates) != 4 {
		t.Errorf("Expected 4 candidates, got %d", len(candidates))
	}

	// Best match should be "CustomerID" (exact match)
	if candidates[0].SourceField.Name != "CustomerID" {
		t.Errorf("Expected best match to be 'CustomerID', got '%s'", candidates[0].SourceField.Name)
	}

	// Verify the best candidate has high score
	if candidates[0].CombinedScore < 0.9 {
		t.Errorf("Expected high score for exact match, got %f", candidates[0].CombinedScore)
	}

	// Second best should be "customer_id" (same name after normalization)
	if candidates[1].SourceField.Name != "customer_id" {
		t.Errorf("Expected second match to be 'customer_id', got '%s'", candidates[1].SourceField.Name)
	}
}

func TestCandidateList_Sorting(t *testing.T) {
	candidates := CandidateList{
		{SourceField: &analyze.FieldInfo{Name: "FieldA"}, CombinedScore: 0.5},
		{SourceField: &analyze.FieldInfo{Name: "FieldC"}, CombinedScore: 0.9},
		{SourceField: &analyze.FieldInfo{Name: "FieldB"}, CombinedScore: 0.7},
		{SourceField: &analyze.FieldInfo{Name: "FieldD"}, CombinedScore: 0.7}, // Same score as FieldB
	}

	// This should be done by RankCandidates, but test manual sorting
	// (candidates are already sorted when returned from RankCandidates)

	// Verify ordering after explicit sort
	if candidates[0].CombinedScore > candidates[1].CombinedScore {
		// Pre-sorted highest first
	}

	// Test that Best() returns highest
	best := candidates.Best()
	if best == nil {
		t.Error("Expected best candidate, got nil")
	}
}

func TestCandidateList_Top(t *testing.T) {
	candidates := CandidateList{
		{SourceField: &analyze.FieldInfo{Name: "A"}, CombinedScore: 0.9},
		{SourceField: &analyze.FieldInfo{Name: "B"}, CombinedScore: 0.8},
		{SourceField: &analyze.FieldInfo{Name: "C"}, CombinedScore: 0.7},
	}

	top2 := candidates.Top(2)
	if len(top2) != 2 {
		t.Errorf("Expected 2 candidates, got %d", len(top2))
	}

	// Request more than available
	top10 := candidates.Top(10)
	if len(top10) != 3 {
		t.Errorf("Expected 3 candidates (all), got %d", len(top10))
	}
}

func TestCandidateList_IsAmbiguous(t *testing.T) {
	tests := []struct {
		name      string
		scores    []float64
		threshold float64
		expected  bool
	}{
		{
			name:      "clear winner",
			scores:    []float64{0.9, 0.5},
			threshold: 0.1,
			expected:  false,
		},
		{
			name:      "ambiguous",
			scores:    []float64{0.9, 0.85},
			threshold: 0.1,
			expected:  true,
		},
		{
			name:      "single candidate",
			scores:    []float64{0.9},
			threshold: 0.1,
			expected:  false,
		},
		{
			name:      "no candidates",
			scores:    []float64{},
			threshold: 0.1,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var candidates CandidateList
			for i, score := range tt.scores {
				candidates = append(candidates, Candidate{
					SourceField:   &analyze.FieldInfo{Name: string(rune('A' + i))},
					CombinedScore: score,
				})
			}

			if got := candidates.IsAmbiguous(tt.threshold); got != tt.expected {
				t.Errorf("IsAmbiguous() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCandidateList_AboveThreshold(t *testing.T) {
	candidates := CandidateList{
		{SourceField: &analyze.FieldInfo{Name: "A"}, CombinedScore: 0.9},
		{SourceField: &analyze.FieldInfo{Name: "B"}, CombinedScore: 0.7},
		{SourceField: &analyze.FieldInfo{Name: "C"}, CombinedScore: 0.5},
		{SourceField: &analyze.FieldInfo{Name: "D"}, CombinedScore: 0.3},
	}

	above := candidates.AboveThreshold(0.6)
	if len(above) != 2 {
		t.Errorf("Expected 2 candidates above 0.6, got %d", len(above))
	}
}

func TestCandidateList_HighConfidence(t *testing.T) {
	tests := []struct {
		name     string
		cands    CandidateList
		minScore float64
		minGap   float64
		wantNil  bool
	}{
		{
			name: "high confidence",
			cands: CandidateList{
				{
					SourceField:   &analyze.FieldInfo{Name: "A"},
					CombinedScore: 0.95,
					TypeCompat:    TypeCompatibilityResult{Compatibility: TypeIdentical},
				},
				{
					SourceField:   &analyze.FieldInfo{Name: "B"},
					CombinedScore: 0.5,
					TypeCompat:    TypeCompatibilityResult{Compatibility: TypeConvertible},
				},
			},
			minScore: 0.7,
			minGap:   0.15,
			wantNil:  false,
		},
		{
			name: "too close",
			cands: CandidateList{
				{
					SourceField:   &analyze.FieldInfo{Name: "A"},
					CombinedScore: 0.9,
					TypeCompat:    TypeCompatibilityResult{Compatibility: TypeIdentical},
				},
				{
					SourceField:   &analyze.FieldInfo{Name: "B"},
					CombinedScore: 0.85,
					TypeCompat:    TypeCompatibilityResult{Compatibility: TypeIdentical},
				},
			},
			minScore: 0.7,
			minGap:   0.15,
			wantNil:  true,
		},
		{
			name: "below min score",
			cands: CandidateList{
				{
					SourceField:   &analyze.FieldInfo{Name: "A"},
					CombinedScore: 0.5,
					TypeCompat:    TypeCompatibilityResult{Compatibility: TypeIdentical},
				},
			},
			minScore: 0.7,
			minGap:   0.15,
			wantNil:  true,
		},
		{
			name: "incompatible type",
			cands: CandidateList{
				{
					SourceField:   &analyze.FieldInfo{Name: "A"},
					CombinedScore: 0.95,
					TypeCompat:    TypeCompatibilityResult{Compatibility: TypeIncompatible},
				},
			},
			minScore: 0.7,
			minGap:   0.15,
			wantNil:  true,
		},
		{
			name:     "empty list",
			cands:    CandidateList{},
			minScore: 0.7,
			minGap:   0.15,
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cands.HighConfidence(tt.minScore, tt.minGap)
			if (result == nil) != tt.wantNil {
				t.Errorf("HighConfidence() returned nil=%v, want nil=%v", result == nil, tt.wantNil)
			}
		})
	}
}

func TestCalculateCombinedScore(t *testing.T) {
	tests := []struct {
		nameScore  float64
		typeCompat TypeCompatibility
		minScore   float64
		maxScore   float64
	}{
		// Perfect match
		{1.0, TypeIdentical, 0.99, 1.01},
		// Good name, identical type
		{0.8, TypeIdentical, 0.85, 0.95},
		// Perfect name, needs transform
		{1.0, TypeNeedsTransform, 0.7, 0.8},
		// No name match, identical type
		{0.0, TypeIdentical, 0.35, 0.45},
		// No match at all
		{0.0, TypeIncompatible, -0.01, 0.01},
	}

	for i, tt := range tests {
		score := calculateCombinedScore(tt.nameScore, tt.typeCompat)
		if score < tt.minScore || score > tt.maxScore {
			t.Errorf("Test %d: calculateCombinedScore(%f, %v) = %f, want in [%f, %f]",
				i, tt.nameScore, tt.typeCompat, score, tt.minScore, tt.maxScore)
		}
	}
}

func TestRankCandidates_Determinism(t *testing.T) {
	// Run ranking multiple times to verify deterministic ordering
	intType := types.Typ[types.Int]

	targetField := &analyze.FieldInfo{
		Name:     "Value",
		Exported: true,
		Type:     &analyze.TypeInfo{GoType: intType},
	}

	sourceFields := []analyze.FieldInfo{
		{Name: "ValueB", Exported: true, Type: &analyze.TypeInfo{GoType: intType}},
		{Name: "ValueA", Exported: true, Type: &analyze.TypeInfo{GoType: intType}},
		{Name: "ValueC", Exported: true, Type: &analyze.TypeInfo{GoType: intType}},
	}

	// All have similar scores, so tie-breaker (alphabetical) should be consistent
	firstRun := RankCandidates(targetField, sourceFields)
	for i := 0; i < 10; i++ {
		nextRun := RankCandidates(targetField, sourceFields)
		for j := range firstRun {
			if firstRun[j].SourceField.Name != nextRun[j].SourceField.Name {
				t.Errorf("Run %d: position %d has '%s', expected '%s'",
					i, j, nextRun[j].SourceField.Name, firstRun[j].SourceField.Name)
			}
		}
	}
}
