package kavach

import (
	"regexp"
	"strings"
	"sync"

	"github.com/gobwas/glob"
)

type Matcher interface {
	Match(pattern, resource string) (bool, error)
}

type ExactMatcher struct{}

func (*ExactMatcher) Match(pattern, resource string) (bool, error) {
	return pattern == resource, nil
}

type RegexMatcher struct {
	cache sync.Map
}

func (rm *RegexMatcher) Match(pattern, resource string) (bool, error) {
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

type PrefixMatcher struct{}

func (*PrefixMatcher) Match(prefix, resource string) (bool, error) {
	return strings.HasPrefix(resource, prefix), nil
}

type GlobMatcher struct {
	cache sync.Map
}

func (rm *GlobMatcher) Match(pattern, resource string) (bool, error) {
	var g glob.Glob
	v, ok := rm.cache.Load(pattern)
	if !ok {
		var err error
		g, err := glob.Compile(pattern)
		if err != nil {
			return false, err
		}
		rm.cache.Store(pattern, g)
	} else {
		g = v.(glob.Glob)
	}

	return g.Match(resource), nil
}
