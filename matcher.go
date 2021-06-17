package vanguard

import (
	"regexp"
	"strings"
	"sync"

	"github.com/srikrsna/glob"
)

// ResourceMatcher is used to match resources.
//
// There are the following strategies already implemented,
// * Exact
// * Prefix
// * Regex
// * Glob
type ResourceMatcher interface {
	MatchResource(has, need string) (bool, error)
}

// ResourceMatcher is used to match permission levels.
//
// There are the following strategies already implemented,
// * Exact
// * Ordered
// * BitMask
type LevelMatcher interface {
	MatchLevel(has, required int64) bool
}

// ExactResourceMatcher matches if both the pattern and resource are exactly equal
type ExactResourceMatcher struct{}

func (*ExactResourceMatcher) MatchResource(pattern, resource string) (bool, error) {
	return pattern == resource, nil
}

// RegexResourceMatcher matches if the resource satisfies the pattern (regex)
// It uses go's std regex library which follows the re2 syntax
type RegexResourceMatcher struct {
	cache sync.Map
}

func (rm *RegexResourceMatcher) MatchResource(pattern, resource string) (bool, error) {
	var expr *regexp.Regexp
	v, ok := rm.cache.Load(pattern)
	if !ok {
		var err error
		expr, err = regexp.Compile(pattern)
		if err != nil {
			return false, err
		}
		rm.cache.Store(pattern, expr)
	} else {
		expr = v.(*regexp.Regexp)
	}

	return expr.MatchString(resource), nil
}

// RegexResourceMatcher matches if the resource has the pattern as prefix
type PrefixResourceMatcher struct{}

func (*PrefixResourceMatcher) MatchResource(prefix, resource string) (bool, error) {
	return strings.HasPrefix(resource, prefix), nil
}

// RegexResourceMatcher matches if the resource satisfies the pattern (glob)
// It uses srikrsna/glob package to compile and match globs. It is documented as follows,
// 
// Match reports whether resource matches the shell pattern.
// The pattern syntax is:
//
//	pattern:
//		{ term }
//	term:
//		'*'         matches any sequence of non-/ characters
//		'**'        matches any sequence of characters
//		'?'         matches any single non-/ character
//		'[' [ '!' ] { character-range } ']'
//		            character class (must be non-empty)
//		c           matches character c (c != '*', '?', '\\', '[')
//		'\\' c      matches character c
//
//	character-range:
//		c           matches character c (c != '\\', '-', ']')
//		'\\' c      matches character c
//		lo '-' hi   matches character c for lo <= c <= hi
//
// Match requires pattern to match all of resource, not just a substring.
// The only possible returned error is ErrBadPattern, when pattern
// is malformed.
//
type GlobResourceMatcher struct {
}

func (rm *GlobResourceMatcher) MatchResource(pattern, resource string) (bool, error) {
	return glob.Match(pattern, resource)
}

// ExactLevelMatcher matches if both the levels are exactly equal
type ExactLevelMatcher struct {
}

func (*ExactLevelMatcher) MatchLevel(has, needs int) bool {
	return has == needs
}

// OrderedLevelMatcher matches if comparision succeeds based on the Asc parameter.
//
// If Asc is false (default), the user needs to have equal or less than the level that is required for an operation i.e. levels behave like ranks
// If Asc is true, the user needs to have equal or greater than the level that is required for an operation
//
// Defaults to Asc false
type OrderedLevelMatcher struct {
	Asc bool
}

func (o *OrderedLevelMatcher) MatchLevel(has, needs int64) bool {
	return (o.Asc && has >= needs) || (!o.Asc && has <= needs)
}

// BitMaskLevelMatcher matches by doing bitwise AND and checking if the user has all the needed bits set.
type BitMaskLevelMatcher struct {
}

func (*BitMaskLevelMatcher) MatchLevel(has, needs int) bool {
	return has&needs == needs
}
