package kavach_test

import (
	"testing"

	"github.com/srikrsna/kavach"
	expb "github.com/srikrsna/kavach/example"
	rlpb "github.com/srikrsna/kavach/kavach"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/proto"
)

const (
	Owner   = 1
	Manager = 5
	Editor  = 10
	Viewer  = 15

	Service = "/example.ExampleService"
	Create  = Service + "/CreateExample"
	Update  = Service + "/UpdateExample"
	Delete  = Service + "/DeleteExample"
	List    = Service + "/ListExample"
	Get     = Service + "/GetExample"
)

type testcase struct {
	Name        string
	Method      string
	Request     proto.Message
	Permissions []*rlpb.Permission
	Allow       bool
}

func (tc *testcase) Evaluate(assert kavach.Kavach, l *zap.Logger) {
	e := assert[tc.Method]
	res, det, err := e.Eval(map[string]interface{}{
		"r": tc.Request,
		"u": tc.Permissions,
	})
	if err != nil {
		l.Fatal("unable to evaluate expr", zap.Error(err), zap.Any("details", det))
	}

	if v, ok := res.Value().(bool); !ok || (v != tc.Allow) {
		l.Fatal("output mismatch", zap.Any("act", res.Value()), zap.Bool("exp", tc.Allow))
	}
}

var Cases = []testcase{
	{
		Name:   "Create_Valid",
		Method: Create,
		Request: &expb.CreateExampleRequest{
			Parent: "/parents/12422",
		},
		Permissions: []*rlpb.Permission{
			{Level: Owner, Resources: []string{"/parents/12422/examples/"}},
		},
		Allow: true,
	},
	{
		Name:   "Create_Inalid",
		Method: Create,
		Request: &expb.CreateExampleRequest{
			Parent: "/parents/12423",
		},
		Permissions: []*rlpb.Permission{
			{Level: Owner, Resources: []string{"/parents/12422/examples/"}},
		},
		Allow: false,
	},
}

func TestAssertions(t *testing.T) {
	l := zaptest.NewLogger(t, zaptest.Level(zapcore.ErrorLevel))
	defer l.Sync()
	assert, err := kavach.NewKavach()
	if err != nil {
		l.Fatal("unable to compile assertions", zap.Error(err))
	}

	for _, tc := range Cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			tc.Evaluate(assert, l)
		})
	}
}

func BenchmarkAssertions(b *testing.B) {
	l := zaptest.NewLogger(b, zaptest.Level(zapcore.ErrorLevel))
	defer l.Sync()
	assert, err := kavach.NewKavach()
	if err != nil {
		l.Fatal("unable to compile assertions", zap.Error(err))
	}

	b.ResetTimer()
	for _, bc := range Cases {
		bc := bc
		b.Run(bc.Name, func(b *testing.B) {
			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					bc.Evaluate(assert, l)
				}
			})
		})
	}
}
