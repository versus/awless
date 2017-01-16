package script

import (
	"errors"
	"fmt"
	"log"
	"reflect"
	"testing"

	"github.com/wallix/awless/script/ast"
	"github.com/wallix/awless/script/driver"
)

func TestRunDriverOnScript(t *testing.T) {
	t.Run("Driver visit expression nodes", func(t *testing.T) {
		s := &Script{&ast.AST{}}

		n := &ast.ExpressionNode{
			Action: "create", Entity: "vpc",
			Params: map[string]interface{}{"count": 1},
		}
		s.Statements = append(s.Statements, n)

		mDriver := &mockDriver{
			action: "create", entity: "vpc",
			expectedParams: map[string]interface{}{"count": 1},
		}

		if err := s.Run(mDriver); err != nil {
			t.Fatal(err)
		}
		if !mDriver.lookupCalled() {
			t.Fatal("driver lookup not called")
		}
	})

	t.Run("Driver visit declaration nodes", func(t *testing.T) {
		s := &Script{&ast.AST{}}

		decl := &ast.DeclarationNode{
			Left: &ast.IdentifierNode{Ident: "myvar"},
			Right: &ast.ExpressionNode{
				Action: "create", Entity: "vpc",
				Params: map[string]interface{}{"count": 1},
			},
		}
		s.Statements = append(s.Statements, decl)

		mDriver := &mockDriver{
			action: "create", entity: "vpc",
			expectedParams: map[string]interface{}{"count": 1},
		}

		if err := s.Run(mDriver); err != nil {
			t.Fatal(err)
		}

		if got, want := decl.Left.Val, "mynewvpc"; got != want {
			t.Fatalf("identifier: got %#v, want %#v", got, want)
		}

		if !mDriver.lookupCalled() {
			t.Fatal("driver lookup not called")
		}
	})
}

func TestResolveTemplate(t *testing.T) {

	t.Run("Holes Resolution", func(t *testing.T) {
		s := &Script{&ast.AST{}}

		expr := &ast.ExpressionNode{
			Holes: map[string]string{"name": "presidentName", "rank": "presidentRank"},
		}
		s.Statements = append(s.Statements, expr)

		decl := &ast.DeclarationNode{
			Right: &ast.ExpressionNode{
				Holes: map[string]string{"age": "presidentAge", "wife": "presidentWife"},
			},
		}
		s.Statements = append(s.Statements, decl)

		fills := map[string]interface{}{
			"presidentName": "trump",
			"presidentRank": 45,
			"presidentAge":  70,
			"presidentWife": "melania",
		}

		s.ResolveTemplate(fills)

		expected := map[string]interface{}{"name": "trump", "rank": 45}
		if got, want := expr.Params, expected; !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if got, want := len(expr.Holes), 0; got != want {
			t.Fatalf("length of holes: got %d, want %d", got, want)
		}

		expected = map[string]interface{}{"age": 70, "wife": "melania"}
		if got, want := decl.Right.Params, expected; !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if got, want := len(decl.Right.Holes), 0; got != want {
			t.Fatalf("length of holes: got %d, want %d", got, want)
		}
	})

	t.Run("Interactive holes resolution", func(t *testing.T) {
		s := &Script{&ast.AST{}}

		expr := &ast.ExpressionNode{
			Holes: map[string]string{"age": "age_of_president", "name": "name_of_president"},
		}
		s.Statements = append(s.Statements, expr)

		each := func(question string) interface{} {
			if question == "Age_of_president" {
				return 70
			}
			if question == "Name_of_president" {
				return "trump"
			}

			return nil
		}

		s.InteractiveResolveTemplate(each)

		expected := map[string]interface{}{"age": 70, "name": "trump"}
		if got, want := expr.Params, expected; !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		if got, want := len(expr.Holes), 0; got != want {
			t.Fatalf("length of holes: got %d, want %d", got, want)
		}
	})
}

type mockDriver struct {
	lookupDone     bool
	action, entity string
	expectedParams map[string]interface{}
}

func (r *mockDriver) lookupCalled() bool {
	defer func() {
		r.lookupDone = false
	}()

	return r.lookupDone
}

func (r *mockDriver) Lookup(lookups ...string) driver.DriverFn {
	r.lookupDone = true
	if lookups[0] == r.action && lookups[1] == r.entity {
		return func(params map[string]interface{}) (interface{}, error) {
			if got, want := params, r.expectedParams; !reflect.DeepEqual(got, want) {
				return nil, fmt.Errorf("[%s %s] params mismatch: expected %v, got %v", r.action, r.entity, got, want)
			}
			return "mynewvpc", nil
		}
	}

	return func(params map[string]interface{}) (interface{}, error) {
		return nil, errors.New("Unexpected lookup fallthrough")
	}
}

func (r *mockDriver) SetLogger(*log.Logger) {}