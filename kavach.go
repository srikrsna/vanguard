package kavach

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter/functions"
	pb "github.com/srikrsna/kavach/kavach"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type Kavach map[string]cel.Program

func NewKavach(opts ...Option) (Kavach, error) {
	var (
		store = Kavach{}
		me    = MultiError{}
		opt   = &Options{
			Roles:           DefaultLevels(),
			ResourceMatcher: &ExactResourceMatcher{},
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

	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			s := services.Get(i)
			methods := s.Methods()
			for j := 0; j < methods.Len(); j++ {
				m := methods.Get(j)
				if m.IsStreamingClient() {
					continue
				}

				exp := proto.GetExtension(m.Options().(*descriptorpb.MethodOptions), pb.E_Assert).(string)
				if exp == "" {
					continue
				}

				fmn := "/" + string(s.FullName()) + "/" + string(m.Name())

				r := m.Input()
				rt, err := protoregistry.GlobalTypes.FindMessageByName(r.FullName())
				if err != nil {
					me = append(me, fmt.Errorf("kavach: unable to find proto type: %s, err: %w", string(r.FullName()), err))
					continue
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
					me = append(me, fmt.Errorf("kavach: unable to create cel env: %w", err))
					continue
				}

				ast, iss := env.Compile(exp)
				if err := iss.Err(); err != nil {
					me = append(me, fmt.Errorf("kavach: unable to parse exp: %w", err))
					continue
				}

				if !proto.Equal(ast.ResultType(), decls.Bool) {
					me = append(me, fmt.Errorf("kavach: assert expression is not a bool, got: %v", ast.ResultType()))
					continue
				}

				prg, err := env.Program(ast, funcs)
				if err != nil {
					me = append(me, fmt.Errorf("kavach: unable to generate eval: %w", err))
					continue
				}

				store[fmn] = prg
			}
		}

		return true
	})

	if len(me) > 0 {
		return nil, me
	}

	return store, nil
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
				ok, err := mf.rm.MatchResource(pr, cr)
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
				ok, err := mf.rm.MatchResource(r, cr)
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

func extractTypes(values []ref.Val) ([]*pb.Permission, int64, []string, ref.Val) {
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

	rr := make([]string, 0, len(vv))
	for _, v := range vv {
		r, ok := v.Value().(string)
		if !ok {
			return nil, -1, nil, types.MaybeNoSuchOverloadErr(v)
		}
		rr = append(rr, r)
	}

	return u, lv, rr, nil
}

type MultiError []error

func (me MultiError) Error() string {
	var sb strings.Builder
	for _, e := range me {
		sb.WriteString(e.Error())
	}
	return sb.String()
}
