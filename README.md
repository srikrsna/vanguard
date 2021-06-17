# Vanguard
[![Go Report Card](https://goreportcard.com/badge/github.com/srikrsna/vanguard)](https://goreportcard.com/report/github.com/srikrsna/vanguard) [![Go Reference](https://pkg.go.dev/badge/github.com/srikrsna/vanguard.svg)](https://pkg.go.dev/github.com/srikrsna/vanguard) ![Tests](https://github.com/srikrsna/vanguard/actions/workflows/go.yml/badge.svg)

Package vanguard provides configurable access control mechanism for gRPC endpoints in Go. Although the same can be applied to any request/response model like the OpenAPI, as of now it only has support for gRPC. It is designed to solve for Restful API architectures. But it can be used pretty much everywhere the concepts hold.

## Concept 
On a high level, it typical for api calls to have the following,

* The one requesting for something to happen - Subject/Client/User
* The something that needs to happen - Action/Task/Method/RPC - 
* The one on which the something is happening - Resource/Object/Entity

For sake of brevity I'll start referring to them as follows from now on,

* User
* Action
* Resource

For every request Vanguard helps you figure out if the **User** can perform an **Action** on a **Resource**.

## Example

At a system level one needs to define a set of Access Levels. Each level is a plain old int64. Vanguard by default provides four simple levels: Owner, Manager, Editor, Viewer. They are obviously ranked in that order. Now let's take a simple CRUD service:

```protobuf
import "vanguard/vanguard.proto";

service PagesService {
  // omitted for brevity

  rpc GetPage(GetPageRequest) returns (Page) {
    option (vanguard.assert) = "u.hasAny(VIEWER, [r.id])";
  }

  // omitted for brevity
}
message GetPageRequest {  
  string id = 1;
}

// omitted for brevity
```

Okay! so you may have noticed that we are defining an rpc option `(vanguard.assert) = '...'`. Specifying this option at method level tells vanguard to only allow access to this if this assertion holds true.

Let's take the assertion for the get method and dig deeper: `u.hasAny(VIEWER, [r.id])`

The syntax we are using is of [cel](https://github.com/google/cel-spec). It is similar to common programming languages and was designed for use cases such as this.

Vanguard gives you certain predefined variables, 
* User - `u`
* Request Message - `r`
* Access Levels as constants - `OWNER`/`VIEWER`/`MANAGER`/`EDITOR` (Modifiable)

In addition to this it also provides certain functions/methods. For example the `hasAny` method on user checks if a user has a specified access level assigned on at least one of the resources. It takes the access level as it's first argument and a list of resource ids as it's second.

So in summary the `u.hasAny(VIEWER, [r.id])` translates to: Allow if the user has Viewer level access to the requested resource.

Thanks to the power of cel these expressions can be as complex as one needs them to be. The only requirement is that the expression must always resolve to a boolean expression. (Don't worry this is type checked ahead of time by vanguard)

Now in our code while somewhere at the beginning of the program,

```go
func main() {
    vg, err := vanguard.NewVanguard()
    if err != nil {
        // handle error
    }
}
```

This automatically parses through all of the grpc services imported into package and compiles all the expressions. It returns an error in the case of one or more compilation errors.

But wait, we haven't asked our vanguard to enforce yet. To do that we just need to add a UnaryInterceptor,
```go
func main() {
    // Initialization code

    pf := ... func(context.Context) ([]*pb.Permission, error) {
        // Extract user info from context
        // Fetch and return permissions 
    }

    vgcept := vanguard.Interceptor(vg, pf, nil)

    // pass vgcept to grpc unary interceptor chain
}

```

The interceptor needs a way to acquire the permissions of the current user. It requires a function that can return all the Access Levels of a user. `pf` is that function.

## Matching

If you look at the get example again, we are only asking for a Viewer level on the resources. Naturally a user with Owner privileges on the resource should also be able to perform the action. One way to go about it is to assign Viewer and other levels whenever Owner is assigned. This way it is guaranteed that an Owner will always have the lower level privileges.

This approach may be straight forward but doesn't scale very well. Instead vanguard provides matching strategies for matching levels with the default being a ordered strategy. Remember that I said these access levels are just an alias for plain old int64? This can be used to order and the levels in ascending and descending fashion. In addition to this Vanguard also offers bit mask based matching strategies.

The same reasoning is valid for matching resources. Let's understand this with and example. Imagine a simple CRUD API for books and pages. Each page belongs to exactly one book. So each page can be identified using something like 'books/1242/pages/76'.

It becomes impractical to give access to all the pages to a particular user. Instead in this case we can again change the matching strategy for resources to something like a glob based matching strategy. Then in the books example a user would be given access to book and it pages with 'books/1242/pages/*'. This would mean the user has access to all the pages of a book.

These particular matching strategies (Ordered for levels, Glob for resources), scale well with Rest architectures.

List of supported strategies are,

### Access Level Matching Strategies

* Exact: The access level should be exactly equal
* Ordered - Ascending: The access level's are ordered in ascending order, i.e. Owner (10) > Viewer (1) 
* Ordered - Descending: The access level's are ordered in descending order, i.e. Owner (1) < Viewer (10) (**Default**) 

### Resource Matching Strategies

* Exact
* Prefix
* Regex
* [Glob](https://pkg.go.dev/github.com/srikrsna/glob) (**Default**)

## Assertion options

Vanguard gives you certain predefined variables, 

* User - `u`
* Request Message - `r`
* Access Levels as constants - `OWNER`/`VIEWER`/`MANAGER`/`EDITOR` (Modifiable)

In addition to this it also provides to methods on `u` that evaluate to a boolean

* hasAny
    * Signature: (int64|Level, [string])
    * True if the user has the given access on at least one of the resource
* hasAll
    * Signature: (int64|Level, [string])
    * True iff the user has the given access on all of the resource

And the full power of cel. Cel has first class support for protobuf messages including the well-known-types.

## Permission Store

The package deliberately avoids providing a mechanism to store access levels against a user. This is left to the developers, as more often than not it largely depends on what model of access control is being used. Vanguard provides low level primitives to build well known access control models such as Role based access control. See the RBAC section about how a Role based access control model can be build on top of vanguard primitives.

## RBAC (Role based access control)

Let's continue the books example. To summarize, books have pages. Now we decided to have RBAC and decided to have the following roles,

* Book Owner
* Book Reader

First let's look at the CRUD, I've omitted the irrelevant parts,
```protobuf
service BookService {

  rpc GetBook(GetBookRequest) returns (Book) {
    option (vanguard.assert) = "u.hasAll(VIEWER, [r.id])";
  }  

  rpc DeleteBook(DeleteBookRequest) returns (google.protobuf.Empty) {
    option (vanguard.assert) = "u.hasAny(MANAGER, [r.name])";
  }
}
```

In this case, the roles would have,

* Book Owner:
    * Structure: { BookId int }
    * Pattern: /books/<book id here>* with OWNER
* Book Reader:
    * Structure: { BookId int }
    * Pattern: /books/<book id here>* with VIEWER

Naturally these roles need to be stored somewhere (database). The pattern can be constructed on demand or can also be stored against each time roles are assigned/removed against a user.

For any use case that you are having a problem with achieving or general suggestions to improve the package, please open an issue.