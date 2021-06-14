package vanguard_test

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/srikrsna/vanguard"
	expb "github.com/srikrsna/vanguard/example"
	pb "github.com/srikrsna/vanguard/vanguard"
	"go.uber.org/zap"
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
	Method      string
	Request     proto.Message
	Permissions []*pb.Permission

	ResourceMatcher vanguard.ResourceMatcher
	LevelMatcher    vanguard.LevelMatcher

	Allow bool
}

func (tc *testcase) Evaluate(assert vanguard.vanguard, l *zap.Logger) {
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

func (tc *testcase) Name() string {
	return fmt.Sprintf("%s_%s_%s_%v",
		strings.TrimSuffix(strings.TrimPrefix(tc.Method, Service+"/"), "Example"),
		strings.TrimSuffix(reflect.ValueOf(tc.LevelMatcher).Type().Elem().Name(), "LevelMatcher"),
		strings.TrimSuffix(reflect.ValueOf(tc.ResourceMatcher).Type().Elem().Name(), "ResourceMatcher"),
		tc.Allow,
	)
}

var Cases = []testcase{
	{
		Method: Create,
		Request: &expb.CreateExampleRequest{
			Parent: "/parents/12422",
		},
		Permissions: []*pb.Permission{
			{Level: Owner, Resources: []string{"/parents/12422/examples/"}},
		},
		ResourceMatcher: &vanguard.ExactResourceMatcher{},
		LevelMatcher:    &vanguard.OrderedLevelMatcher{},
		Allow:           true,
	},
	{
		Method: Create,
		Request: &expb.CreateExampleRequest{
			Parent: "/parents/12422",
		},
		Permissions: []*pb.Permission{
			{Level: Owner, Resources: []string{"/parents/12422/examples/"}},
		},
		ResourceMatcher: &vanguard.PrefixResourceMatcher{},
		LevelMatcher:    &vanguard.OrderedLevelMatcher{},
		Allow:           true,
	},
	{
		Method: Create,
		Request: &expb.CreateExampleRequest{
			Parent: "/parents/12422",
		},
		Permissions: []*pb.Permission{
			{Level: Owner, Resources: []string{"/parents/12422/*"}},
		},
		ResourceMatcher: &vanguard.GlobResourceMatcher{},
		LevelMatcher:    &vanguard.OrderedLevelMatcher{},
		Allow:           true,
	},
	{
		Method: Create,
		Request: &expb.CreateExampleRequest{
			Parent: "/parents/12422",
		},
		Permissions: []*pb.Permission{
			{Level: Owner, Resources: []string{"/parents/12422/.*"}},
		},
		ResourceMatcher: &vanguard.RegexResourceMatcher{},
		LevelMatcher:    &vanguard.OrderedLevelMatcher{},
		Allow:           true,
	},
	{
		Method: Create,
		Request: &expb.CreateExampleRequest{
			Parent: "/parents/12423",
		},
		Permissions: []*pb.Permission{
			{Level: Owner, Resources: []string{"/parents/12422/examples/"}},
		},
		ResourceMatcher: &vanguard.ExactResourceMatcher{},
		LevelMatcher:    &vanguard.OrderedLevelMatcher{},
		Allow:           false,
	},
	{
		Method: Create,
		Request: &expb.CreateExampleRequest{
			Parent: "/parents/12423",
		},
		Permissions: []*pb.Permission{
			{Level: Owner, Resources: []string{"/parents/12422/examples/"}},
		},
		ResourceMatcher: &vanguard.PrefixResourceMatcher{},
		LevelMatcher:    &vanguard.OrderedLevelMatcher{},
		Allow:           false,
	},
	{
		Method: Create,
		Request: &expb.CreateExampleRequest{
			Parent: "/parents/12423",
		},
		Permissions: []*pb.Permission{
			{Level: Owner, Resources: []string{"/parents/12422/*"}},
		},
		ResourceMatcher: &vanguard.GlobResourceMatcher{},
		LevelMatcher:    &vanguard.OrderedLevelMatcher{},
		Allow:           false,
	},
	{
		Method: Create,
		Request: &expb.CreateExampleRequest{
			Parent: "/parents/12423",
		},
		Permissions: []*pb.Permission{
			{Level: Owner, Resources: []string{"/parents/12422/.*"}},
		},
		ResourceMatcher: &vanguard.RegexResourceMatcher{},
		LevelMatcher:    &vanguard.OrderedLevelMatcher{},
		Allow:           false,
	},
}
