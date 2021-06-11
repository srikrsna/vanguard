package kavach_test

import (
	"testing"

	"github.com/srikrsna/kavach"
	expb "github.com/srikrsna/kavach/example"
	rlpb "github.com/srikrsna/kavach/kavach"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

func TestPreprocess(t *testing.T) {
	l := zaptest.NewLogger(t, zaptest.Level(zapcore.ErrorLevel))
	defer l.Sync()
	assert, err := kavach.NewKavach()
	if err != nil {
		l.Fatal("unable to compile assertions", zap.Error(err))
	}

	e := assert["/example.ExampleService/CreateExample"]
	res, det, err := e.Eval(map[string]interface{}{
		"r": &expb.CreateExampleRequest{
			Parent: "/parents/12422",
		},
		"u": []*rlpb.Permission{
			{
				Level:     1, // VIEWER
				Resources: []string{"/parents/12422/examples/"},
			},
		},
	})
	if err != nil {
		l.Fatal("unable to evaluate expr", zap.Error(err), zap.Any("details", det))
	}

	if v, ok := res.Value().(bool); !ok || !v {
		l.Fatal("output mismatch", zap.Any("output", res.Value()))
	}
}
