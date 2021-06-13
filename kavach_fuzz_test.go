//go:build gofuzzbeta
// +build gofuzzbeta

package kavach_test

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/srikrsna/kavach"
	expb "github.com/srikrsna/kavach/example"
	exfuzz "github.com/srikrsna/kavach/example/fuzz"
	kavachpb "github.com/srikrsna/kavach/kavach"
	kavachfz "github.com/srikrsna/kavach/kavach/fuzz"
	zapp "github.com/srikrsna/zapproto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

func FuzzEval(f *testing.F) {
	l := zaptest.NewLogger(f, zaptest.Level(zapcore.ErrorLevel))
	defer l.Sync()
	assert, err := kavach.NewKavach()
	if err != nil {
		l.Fatal("unable to compile assertions", zap.Error(err))
	}

	// Regressions
	f.Add([]byte("\xe4\x89Wh\x92\xbf\xab"))
	f.Add([]byte("\x89\x00\x00\u007f\xffF"))

	f.Fuzz(func(t *testing.T, data []byte) {
		t.Parallel()
		f := fuzz.NewFromGoFuzz(data).Funcs(exfuzz.FuzzFuncs()...).Funcs(kavachfz.FuzzFuncs()...)

		var (
			u []*kavachpb.Permission
			r expb.CreateExampleRequest
		)
		f.Fuzz(&u)
		f.Fuzz(&r)

		res, det, err := assert[Create].Eval(map[string]interface{}{
			"r": &r,
			"u": u,
		})
		if err != nil {
			l.Fatal("error evaluating", zap.Error(err), zap.Any("details", det), zapp.P("r", &r), zap.Any("u", u))
		}

		if v, ok := res.Value().(bool); !ok || (v != false) {
			l.Fatal("output mismatch", zap.Any("act", res.Value()), zap.Bool("exp", false), zapp.P("r", &r), zap.Any("u", u))
		}
	})
}
