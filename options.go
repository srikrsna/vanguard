package vanguard

const (
	LevelOwner   = 1
	LevelManager = 5
	LevelEditor  = 10
	LevelViewer  = 15
)

// Level defines a permission level. Name can be used as is in the assert
// expressions. They are substituted with their corresponding Value
type Level struct {
	Name  string
	Value int64
}

// DefaultLevels are only a placeholder, They can be used in a production system.
// But typically they are overridden.
//
// Look at `WithRoles` to override them
func DefaultLevels() []Level {
	return []Level{
		{Name: "OWNER", Value: 1},
		{Name: "MANAGER", Value: 5},
		{Name: "EDITOR", Value: 10},
		{Name: "VIEWER", Value: 15},
	}
}

type options struct {
	Roles []Level

	ResourceMatcher ResourceMatcher
	LevelMatcher    LevelMatcher
}

type option func(*options)

// WithRoles is used to replace the base set of roles that can be used in the assert expressions
func WithRoles(rl []Level) option {
	return func(o *options) {
		o.Roles = rl
	}
}

// WithResourceMatcher can be used to replace the resource matching strategies
//
// List of available options: Exact, Prefix, Regex, and Glob
func WithResourceMatcher(m ResourceMatcher) option {
	return func(o *options) {
		o.ResourceMatcher = m
	}
}

// WithLevelMatcher can be used to replace the level matching strategies
//
// List of available options: Exact, Ordered, and BitMask
func WithLevelMatcher(m LevelMatcher) option {
	return func(o *options) {
		o.LevelMatcher = m
	}
}
