package fuzzy

// Match is the outcome of searching a candidate list.
type Match[T any] struct {
	Item  T
	Score int
}

// KeyFunc extracts the text a candidate is compared by.
type KeyFunc[T any] func(T) string

// Identity is the KeyFunc for candidates that are already strings.
func Identity(s string) string { return s }

// Best returns the candidate whose key scores highest against query. The first
// candidate wins a tie. ok is false when no candidate reaches minScore.
func Best[T any](query string, candidates []T, key KeyFunc[T], minScore int) (match Match[T], ok bool) {
	for _, candidate := range candidates {
		if score := Ratio(query, key(candidate)); score > match.Score {
			match = Match[T]{Item: candidate, Score: score}
		}
	}
	if match.Score < minScore || match.Score == 0 {
		return Match[T]{}, false
	}
	return match, true
}
