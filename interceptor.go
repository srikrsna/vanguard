package vanguard

import (
	"context"
	"log"
	"sync"

	"github.com/google/cel-go/interpreter"
	pb "github.com/srikrsna/vanguard/vanguard"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrorLogger func(v ...interface{})

// PermissionsFunc is used to retreive the permissions of the current user.
// The context passed is an incoming grpc context.
//
// If it returns an error, it will be returned to the user.
type PermissionsFunc func(context.Context) ([]*pb.Permission, error)

type InterceptorOptions struct {
	Skip        bool
	ErrorLogger ErrorLogger
}

// Interceptor is grpc UnaryServerInterceptor that asserts that a caller has permission to access the endpoints.
// PermissionsFunc is used  to retreive the permissions of the current user
func Interceptor(store Vanguard, pf PermissionsFunc, opt *InterceptorOptions) grpc.UnaryServerInterceptor {
	if opt.Skip {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			return handler(ctx, req)
		}
	}

	if opt.ErrorLogger == nil {
		opt.ErrorLogger = log.Println
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		assert, ok := store[info.FullMethod]
		if !ok {
			return handler(ctx, req)
		}

		perms, err := pf(ctx)
		if err != nil {
			return nil, err
		}

		vars := varPool.Get().(*activation)
		defer varPool.Put(vars)

		vars.R = req
		vars.U = perms

		v, _, err := assert.Eval(vars)
		if err != nil {
			opt.ErrorLogger("vanguard: unable to evaluate access assertions, most likely a bug in vanguard, please open an issue: %v", err)
			return nil, status.Error(codes.Unknown, "Unknown error")
		}

		allow, ok := v.Value().(bool)
		if !ok {
			opt.ErrorLogger("vanguard: unable to evaluate access assertions to bool, most likely a bug in vanguard, please open an issue: type: %[0]T, value: %[0]v", v.Value())
			return nil, status.Error(codes.Unknown, "Unknown error")
		}

		if !allow {
			return nil, status.Error(codes.PermissionDenied, codes.PermissionDenied.String())
		}

		return handler(ctx, req)
	}
}

var varPool = sync.Pool{
	New: func() interface{} {
		return new(activation)
	},
}

var _ interpreter.Activation = (*activation)(nil)

type activation struct {
	R interface{}
	U []*pb.Permission
}

func (a *activation) ResolveName(name string) (interface{}, bool) {
	switch name {
	case "r":
		return a.R, true
	case "u":
		return a.U, true
	default:
		return nil, false
	}
}

func (a *activation) Parent() interpreter.Activation { return nil }
