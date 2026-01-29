// Package match provides name normalization, Levenshtein distance calculation,
// type compatibility scoring, and candidate ranking for field matching.
//
// Key functions:
//   - NormalizeIdent: normalizes identifiers for fuzzy matching
//   - Levenshtein: computes edit distance between strings
//   - ScoreTypeCompatibility: scores type compatibility using go/types
//   - RankCandidates: ranks potential field mappings
package match
