//go:build gofuzzbeta
// +build gofuzzbeta

package vanguard_test

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/srikrsna/vanguard"
	expb "github.com/srikrsna/vanguard/example"
	exfuzz "github.com/srikrsna/vanguard/example/fuzz"
	vanguardfz "github.com/srikrsna/vanguard/vanguard/fuzz"
	zapp "github.com/srikrsna/zapproto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/proto"
)

func FuzzEval(f *testing.F) {
	l := zaptest.NewLogger(f, zaptest.Level(zapcore.ErrorLevel))
	defer l.Sync()
	assert, err := vanguard.NewVanguard()
	if err != nil {
		l.Fatal("unable to compile assertions", zap.Error(err))
	}

	for _, fc := range Cases {
		f.Fuzz(func(t *testing.T, data []byte) {
			t.Parallel()
			f := fuzz.NewFromGoFuzz(data).Funcs(exfuzz.FuzzFuncs()...).Funcs(vanguardfz.FuzzFuncs()...)

			var r proto.Message
			switch fc.Request.(type) {
			case *expb.CreateExampleRequest:
				x := expb.CreateExampleRequest{}
				f.Fuzz(&x)
				r = &x
			case *expb.GetExampleRequest:
				x := expb.GetExampleRequest{}
				f.Fuzz(&x)
				r = &x
			case *expb.ListExamplesRequest:
				x := expb.ListExamplesRequest{}
				f.Fuzz(&x)
				r = &x
			case *expb.DeleteExampleRequest:
				x := expb.DeleteExampleRequest{}
				f.Fuzz(&x)
				r = &x
			case *expb.UpdateExampleRequest:
				x := expb.UpdateExampleRequest{}
				f.Fuzz(&x)
				r = &x
			default:
				t.Skip()
			}

			u := fc.Permissions
			res, det, err := assert[Create].Eval(map[string]interface{}{
				"r": r,
				"u": u,
			})
			if err != nil {
				l.Fatal("error evaluating", zap.Error(err), zap.Any("details", det), zapp.P("r", r), zap.Any("u", u))
			}

			if v, ok := res.Value().(bool); !ok || (v != false) {
				l.Fatal("output mismatch", zap.Any("act", res.Value()), zap.Bool("exp", false), zapp.P("r", r), zap.Any("u", u))
			}
		})
	}
}
