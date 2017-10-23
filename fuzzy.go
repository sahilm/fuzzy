/*
Package fuzzy provides fuzzy string matching optimized
for filenames and code symbols in the style of Sublime Text,
VSCode, IntelliJ IDEA et al.
*/
package fuzzy

import (
	"sort"
	"strings"
)

// Match represents a matched string.
type Match struct {
	// The matched string.
	Str string
	// The index of the matched string in the supplied slice.
	Index int
	// The indexes of matched characters. Useful for highlighting matches.
	MatchedIndexes []int
	// Marker to identify if the Match has been initialized
	initialized bool
	// Score used to rank matches
	score int
}

const (
	firstCharMatchBonus            = 10
	matchFollowingSeparatorBonus   = 20
	camelCaseMatchBonus            = 20
	adjacentMatchBonus             = 5
	unmatchedLeadingCharPenalty    = -3
	maxUnmatchedLeadingCharPenalty = -9
)

var separators = []string{"/", "-", "_", " ", "."}

/*
Find looks up pattern in data and returns matches
in descending order of match quality. Match quality
is determined by a set of bonus and penalty rules.

The following types of matches apply a bonus:

* The first character in the pattern matches the first character in the match string.

* The matched character is camel cased.

* The matched character follows a separator such as an underscore character.

* The matched character is adjacent to a previous match.

Penalties are applied for every character in the search string that wasn't matched and all leading
characters upto the first match.
*/
func Find(pattern string, data []string) []Match {
	if len(pattern) == 0 {
		return []Match{}
	}
	matches := make([]Match, 0)
	for i := 0; i < len(data); i++ {
		candidate := strings.Split(data[i], "")
		match := Match{
			MatchedIndexes: make([]int, 0),
		}
		var score int
		patternIndex := 0
		bestScore := -1
		matchedIndex := -1
		currAdjacentMatchBonus := 0
		for j := 0; j < len(candidate); j++ {
			c := strings.ToLower(candidate[j])
			p := strings.ToLower(string(pattern[patternIndex]))
			if p == c {
				// avoid repeatedly setting params that apply to the whole match
				if !match.initialized {
					match.Str = data[i]
					match.Index = i
					match.initialized = true
				}
				score = 0
				if j == 0 {
					score += firstCharMatchBonus
				}
				score += camelCaseBonus(j, candidate)
				score += separatorBonus(j, candidate)
				if len(match.MatchedIndexes) > 0 {
					lastMatch := match.MatchedIndexes[len(match.MatchedIndexes)-1]
					score += adjacentCharBonus(j, lastMatch, currAdjacentMatchBonus)
					// adjacent matches are incremental and keep increasing based on previous adjacent matches
					// thus we need to maintain the current match bonus
					currAdjacentMatchBonus += adjacentCharBonus(j, lastMatch, currAdjacentMatchBonus)
				}
				if score > bestScore {
					bestScore = score
					matchedIndex = j
				}
			}
			nextp := ""
			if patternIndex < len(pattern)-1 {
				nextp = strings.ToLower(string(pattern[patternIndex+1]))
			}
			nextc := ""
			if j < len(candidate)-1 {
				nextc = strings.ToLower(candidate[j+1])
			}
			// We apply the best score when we have the next match coming up or when the search string has ended.
			// Tracking when the next match is coming up allows us to exhaustively find the best match and not necessarily
			// the first match.
			// For example given the pattern "tk" and search string "The Black Knight", exhaustively matching allows us
			// to match the second k thus giving this string a higher score.
			if nextp == nextc || j == len(candidate)-1 {
				if matchedIndex > -1 {
					if len(match.MatchedIndexes) == 0 {
						penalty := matchedIndex * unmatchedLeadingCharPenalty
						if penalty < 0 {
							bestScore += max(penalty, maxUnmatchedLeadingCharPenalty)
						}
					}
					match.score += bestScore
					match.MatchedIndexes = append(match.MatchedIndexes, matchedIndex)
					score = 0
					bestScore = -1
					patternIndex++
				}
			}
		}
		// apply penalty for each unmatched character
		penalty := len(match.MatchedIndexes) - len(data[i])
		match.score += penalty
		if len(match.MatchedIndexes) == len(pattern) {
			matches = insertMatch(matches, match)
		}
	}
	return matches
}

func insertMatch(matches []Match, match Match) []Match {
	i := sort.Search(len(matches), func(i int) bool {
		return matches[i].score <= match.score
	})
	matches = append(matches, Match{})
	copy(matches[i+1:], matches[i:])
	matches[i] = match
	return matches
}

func separatorBonus(i int, s []string) int {
	if i == 0 {
		return 0
	}
	if isSeparator(s[i-1]) {
		return matchFollowingSeparatorBonus
	}
	return 0
}

func camelCaseBonus(i int, s []string) int {
	if i == 0 {
		return 0
	}
	if isLowerCase(s[i-1]) && isUpperCase(s[i]) {
		return camelCaseMatchBonus
	}
	return 0
}

func adjacentCharBonus(i int, lastMatch int, currentBonus int) int {
	if lastMatch == i-1 {
		return currentBonus*2 + adjacentMatchBonus
	}
	return 0
}

func isSeparator(s string) bool {
	for _, sep := range separators {
		if s == sep {
			return true
		}
	}
	return false
}

func isUpperCase(s string) bool {
	return s != strings.ToLower(s)
}

func isLowerCase(s string) bool {
	return s != strings.ToUpper(s)
}

func max(x int, y int) int {
	if x > y {
		return x
	}
	return y
}
