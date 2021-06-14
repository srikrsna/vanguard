package vanguard

import (
	"regexp"
	"strings"
	"sync"

	"github.com/gobwas/glob"
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
// It uses gobwas/glob package to compile and match globs.
type GlobResourceMatcher struct {
	cache sync.Map
}

func (rm *GlobResourceMatcher) MatchResource(pattern, resource string) (bool, error) {
	var g glob.Glob
	v, ok := rm.cache.Load(pattern)
	if !ok {
		var err error
		g, err = glob.Compile(pattern)
		if err != nil {
			return false, err
		}
		rm.cache.Store(pattern, g)
	} else {
		g = v.(glob.Glob)
	}

	return g.Match(resource), nil
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
