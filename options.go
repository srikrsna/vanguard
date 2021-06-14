package vanguard

const (
	LevelOwner   = 1
	LevelManager = 5
	LevelEditor  = 10
	LevelViewer  = 15
)

type Level struct {
	Name  string
	Value int64
}

func DefaultLevels() []Level {
	return []Level{
		{Name: "OWNER", Value: 1},
		{Name: "MANAGER", Value: 5},
		{Name: "EDITOR", Value: 10},
		{Name: "VIEWER", Value: 15},
	}
}

type Options struct {
	Roles []Level

	ResourceMatcher ResourceMatcher
	LevelMatcher    LevelMatcher
}

type Option func(*Options)

func WithRoles(rl []Level) Option {
	return func(o *Options) {
		o.Roles = rl
	}
}

func WithResourceMatcher(m ResourceMatcher) Option {
	return func(o *Options) {
		o.ResourceMatcher = m
	}
}

func WithLevelMatcher(m LevelMatcher) Option {
	return func(o *Options) {
		o.LevelMatcher = m
	}
}
