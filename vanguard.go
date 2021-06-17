package vanguard

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter/functions"
	pb "github.com/srikrsna/vanguard/vanguard"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type Permission = pb.Permission

// Vanguard holds all the compiled assert expressions against the fully qualified
// method name.
//
// Example for key: /package.Service/Method
// Look at `NewVanguard` to see how it can be created
type Vanguard map[string]cel.Program

// NewVanguard reads all the proto files that are imported in the calling module and
// compiles vanguard's assert statements.
//
// See Options for various ways it can be tweaked.
func NewVanguard(opts ...option) (Vanguard, error) {
	var (
		store = Vanguard{}
		me    = MultiError{}
		opt   = &options{
			Roles:           DefaultLevels(),
			ResourceMatcher: &GlobResourceMatcher{},
			LevelMatcher:    &OrderedLevelMatcher{},
		}
	)

	for _, o := range opts {
		o(opt)
	}

	// Global Types
	permSliceType := decls.NewListType(decls.NewObjectType(string((&pb.Permission{}).ProtoReflect().Descriptor().FullName())))
	var gds []*exprpb.Decl
	roleType := decls.NewPrimitiveType(exprpb.Type_INT64)
	for _, r := range opt.Roles {
		d := decls.NewConst(r.Name, roleType, &exprpb.Constant{
			ConstantKind: &exprpb.Constant_Int64Value{Int64Value: r.Value},
		})
		gds = append(gds, d)
	}
	gds = append(gds, decls.NewVar("u", permSliceType))

	gds = append(gds,
		// Functions
		decls.NewFunction(
			"hasAny",
			decls.NewInstanceOverload(
				"user_any_level_resources",
				[]*exprpb.Type{
					permSliceType,
					roleType,
					decls.NewListType(decls.String),
				},
				decls.Bool,
			),
		),
		decls.NewFunction(
			"hasAll",
			decls.NewInstanceOverload(
				"user_all_level_resources",
				[]*exprpb.Type{
					permSliceType,
					roleType,
					decls.NewListType(decls.String),
				},
				decls.Bool,
			),
		),
	)

	mf := matchFuncs{rm: opt.ResourceMatcher, lm: opt.LevelMatcher}

	funcs := cel.Functions(
		&functions.Overload{
			Operator: "user_any_level_resources",
			Function: mf.any,
		},
		&functions.Overload{
			Operator: "user_all_level_resources",
			Function: mf.all,
		},
	)

	type result struct {
		Err  error
		Prg  cel.Program
		Name string
	}
	results := make(chan *result)
	count := 0

	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			s := services.Get(i)
			methods := s.Methods()
			for j := 0; j < methods.Len(); j++ {
				m := methods.Get(j)
				count++
				go func() {
					prg, err := compile(s, m, gds, funcs)
					results <- &result{
						Prg:  prg,
						Name: "/" + string(s.FullName()) + "/" + string(m.Name()),
						Err:  err,
					}
				}()
			}
		}

		return true
	})

	for i := 0; i < count; i++ {
		res := <-results
		if res.Err != nil {
			if res.Err == errSkip {
				continue
			}
			me = append(me, res.Err)
			continue
		}

		store[res.Name] = res.Prg
	}

	if len(me) > 0 {
		return nil, me
	}

	return store, nil
}

var errSkip = errors.New("skip error")

func compile(
	s protoreflect.ServiceDescriptor,
	m protoreflect.MethodDescriptor,
	gds []*exprpb.Decl,
	funcs ...cel.ProgramOption,
) (cel.Program, error) {
	if m.IsStreamingClient() {
		return nil, errSkip
	}

	exp := proto.GetExtension(m.Options(), pb.E_Assert).(string)
	if exp == "" {
		return nil, errSkip
	}

	r := m.Input()
	rt, err := protoregistry.GlobalTypes.FindMessageByName(r.FullName())
	if err != nil {
		return nil, fmt.Errorf("vanguard: unable to find proto type: %s, err: %w", string(r.FullName()), err)
	}

	env, err := cel.NewEnv(
		cel.Types(
			(*pb.Permission)(nil),
		),
		cel.Types(
			rt.New().Interface(),
		),
		cel.Declarations(
			gds...,
		),
		cel.Declarations(
			decls.NewVar(
				"r",
				decls.NewObjectType(string(r.FullName())),
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("vanguard: unable to create cel env: %w", err)
	}

	ast, iss := env.Compile(exp)
	if err := iss.Err(); err != nil {
		return nil, fmt.Errorf("vanguard: unable to parse exp: %w", err)
	}

	if !proto.Equal(ast.ResultType(), decls.Bool) {
		return nil, fmt.Errorf("vanguard: assert expression is not a bool, got: %v", ast.ResultType())
	}

	prg, err := env.Program(ast, funcs...)
	if err != nil {
		return nil, fmt.Errorf("vanguard: unable to generate eval: %w", err)
	}

	return prg, nil
}

type matchFuncs struct {
	rm ResourceMatcher
	lm LevelMatcher
}

func (mf matchFuncs) any(values ...ref.Val) ref.Val {
	permissions, pl, rr, err := extractTypes(values)
	if err != nil {
		return err
	}

	for _, perm := range permissions {
		if perm == nil {
			continue
		}

		if !mf.lm.MatchLevel(perm.Level, pl) {
			continue
		}

		for _, pr := range perm.Resources {
			for _, cr := range rr {
				ok, err := mf.rm.MatchResource(pr, cr.Value().(string))
				if err != nil {
					return types.NewErr(err.Error())
				} else if ok {
					return types.True
				}
			}
		}
	}

	return types.False
}

func (mf matchFuncs) all(values ...ref.Val) ref.Val {
	permissions, pl, rr, err := extractTypes(values)
	if err != nil {
		return err
	}

	for _, cr := range rr {
		found := false
	outer:
		for _, perm := range permissions {
			if perm == nil {
				continue
			}

			if !mf.lm.MatchLevel(perm.Level, pl) {
				continue
			}

			for _, r := range perm.Resources {
				ok, err := mf.rm.MatchResource(r, cr.Value().(string))
				if err != nil {
					return types.NewErr(err.Error())
				} else if ok {
					found = true
					break outer
				}
			}
		}
		if !found {
			return types.False
		}
	}

	return types.True
}

func extractTypes(values []ref.Val) ([]*pb.Permission, int64, []ref.Val, ref.Val) {
	if len(values) != 3 {
		return nil, -1, nil, types.NoSuchOverloadErr()
	}

	u, ok := values[0].Value().([]*pb.Permission)
	if !ok {
		return nil, -1, nil, types.MaybeNoSuchOverloadErr(values[0])
	}

	lv, ok := values[1].Value().(int64)
	if !ok {
		return nil, -1, nil, types.MaybeNoSuchOverloadErr(values[1])
	}

	vv, ok := values[2].Value().([]ref.Val)
	if !ok {
		return nil, -1, nil, types.MaybeNoSuchOverloadErr(values[2])
	}

	return u, lv, vv, nil
}

type MultiError []error

func (me MultiError) Error() string {
	var sb strings.Builder
	for _, e := range me {
		sb.WriteString(e.Error())
	}
	return sb.String()
}
