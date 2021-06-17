package vanguard_test

import (
	"testing"

	"github.com/srikrsna/vanguard"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

func TestAssertions(t *testing.T) {
	l := zaptest.NewLogger(t, zaptest.Level(zapcore.ErrorLevel))
	defer l.Sync()

	for _, tc := range Cases {
		tc := tc
		t.Run(tc.Name(), func(t *testing.T) {
			t.Parallel()
			assert, err := vanguard.NewVanguard(
				vanguard.WithLevelMatcher(tc.LevelMatcher),
				vanguard.WithResourceMatcher(tc.ResourceMatcher),
			)
			if err != nil {
				l.Fatal("unable to compile assertions", zap.Error(err))
			}
			tc.Evaluate(assert, l)
		})
	}
}

func BenchmarkAssertions(b *testing.B) {
	l := zaptest.NewLogger(b, zaptest.Level(zapcore.ErrorLevel))
	defer l.Sync()

	b.ResetTimer()
	for _, bc := range Cases {
		bc := bc
		b.Run(bc.Name(), func(b *testing.B) {
			assert, err := vanguard.NewVanguard(
				vanguard.WithLevelMatcher(bc.LevelMatcher),
				vanguard.WithResourceMatcher(bc.ResourceMatcher),
			)
			if err != nil {
				l.Fatal("unable to compile assertions", zap.Error(err))
			}
			b.ResetTimer()
			b.ReportAllocs()
			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					bc.Evaluate(assert, l)
				}
			})
		})
	}
}
