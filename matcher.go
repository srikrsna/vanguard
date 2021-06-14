package vanguard

import (
	"regexp"
	"strings"
	"sync"

	"github.com/gobwas/glob"
)

type ResourceMatcher interface {
	MatchResource(has, need string) (bool, error)
}

type LevelMatcher interface {
	MatchLevel(has, required int64) bool
}

type ExactResourceMatcher struct{}

func (*ExactResourceMatcher) MatchResource(pattern, resource string) (bool, error) {
	return pattern == resource, nil
}

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

type PrefixResourceMatcher struct{}

func (*PrefixResourceMatcher) MatchResource(prefix, resource string) (bool, error) {
	return strings.HasPrefix(resource, prefix), nil
}

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

type ExactLevelMatcher struct {
}

func (*ExactLevelMatcher) MatchLevel(has, needs int) bool {
	return has == needs
}

type OrderedLevelMatcher struct {
	Asc bool
}

func (o *OrderedLevelMatcher) MatchLevel(has, needs int64) bool {
	return (o.Asc && has >= needs) || (!o.Asc && has <= needs)
}

type BitMaskLevelMatcher struct {
}

func (*BitMaskLevelMatcher) MatchLevel(has, needs int) bool {
	return has&needs == needs
}
