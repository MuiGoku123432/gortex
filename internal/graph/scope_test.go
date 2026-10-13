package graph

import (
	"reflect"
	"strings"
	"testing"
)

func TestScope_NewScope(t *testing.T) {
	scope, err := NewScope(" ws ", " p ", []string{" b ", "a", "b"})
	if err != nil {
		t.Fatalf("NewScope: %v", err)
	}
	want := Scope{Workspace: "ws", Project: "p", Repositories: []string{"a", "b"}}
	if !reflect.DeepEqual(scope, want) {
		t.Fatalf("NewScope = %#v, want %#v", scope, want)
	}

	for _, repos := range [][]string{nil, {}} {
		if _, err := NewScope("ws", "p", repos); err == nil || !strings.Contains(err.Error(), "repository allow-set is required") {
			t.Fatalf("NewScope(%#v) error = %v, want repository allow-set is required", repos, err)
		}
	}

	if _, err := NewScope("ws", "p", []string{"a", " "}); err == nil || !strings.Contains(err.Error(), "repository allow-set contains an empty value") {
		t.Fatalf("NewScope blank entry error = %v, want repository allow-set contains an empty value", err)
	}
}

func TestScope_RequireWorkspaceProject(t *testing.T) {
	for _, scope := range []Scope{
		{Project: "p", Repositories: []string{"a"}},
		{Workspace: "ws", Repositories: []string{"a"}},
	} {
		if err := scope.RequireWorkspaceProject(); err == nil || !strings.Contains(err.Error(), "workspace and project are required") {
			t.Fatalf("RequireWorkspaceProject(%#v) = %v, want workspace and project are required", scope, err)
		}
	}
	if err := (Scope{Workspace: "ws", Project: "p", Repositories: []string{"a"}}).RequireWorkspaceProject(); err != nil {
		t.Fatalf("RequireWorkspaceProject with both set = %v, want nil", err)
	}
}

func TestScope_Allows(t *testing.T) {
	scope := Scope{Repositories: []string{"a", "b"}}
	cases := []struct {
		name  string
		scope Scope
		node  *Node
		want  bool
	}{
		{name: "nil node", scope: scope, node: nil, want: false},
		{name: "empty prefix", scope: scope, node: &Node{ID: "x.go::X"}, want: false},
		{name: "outside the set", scope: scope, node: &Node{ID: "c/x.go::X", RepoPrefix: "c"}, want: false},
		{name: "zero scope", scope: Scope{}, node: &Node{ID: "a/x.go::X", RepoPrefix: "a"}, want: false},
		{name: "in the set", scope: scope, node: &Node{ID: "a/x.go::X", RepoPrefix: "a"}, want: true},
		{
			name:  "workspace mismatch",
			scope: Scope{Workspace: "ws", Repositories: []string{"a"}},
			node:  &Node{ID: "a/x.go::X", RepoPrefix: "a", WorkspaceID: "other"},
			want:  false,
		},
		{
			name:  "workspace match",
			scope: Scope{Workspace: "ws", Repositories: []string{"a"}},
			node:  &Node{ID: "a/x.go::X", RepoPrefix: "a", WorkspaceID: "ws"},
			want:  true,
		},
		{
			name:  "project mismatch",
			scope: Scope{Project: "p", Repositories: []string{"a"}},
			node:  &Node{ID: "a/x.go::X", RepoPrefix: "a", ProjectID: "other"},
			want:  false,
		},
		{
			name:  "project match",
			scope: Scope{Project: "p", Repositories: []string{"a"}},
			node:  &Node{ID: "a/x.go::X", RepoPrefix: "a", ProjectID: "p"},
			want:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.scope.Allows(tc.node); got != tc.want {
				t.Fatalf("Allows = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestScope_AllowsRepo(t *testing.T) {
	scope := Scope{Repositories: []string{"a", "b"}}
	if scope.AllowsRepo("") {
		t.Fatal(`AllowsRepo("") = true, want false`)
	}
	if !scope.AllowsRepo("b") {
		t.Fatal(`AllowsRepo("b") = false, want true`)
	}
	if scope.AllowsRepo("c") {
		t.Fatal(`AllowsRepo("c") = true, want false`)
	}
	if (Scope{}).AllowsRepo("a") {
		t.Fatal(`zero Scope AllowsRepo("a") = true, want false`)
	}
}
