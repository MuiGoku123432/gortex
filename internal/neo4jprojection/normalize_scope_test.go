package neo4jprojection

import (
	"reflect"
	"testing"
)

// TestNormalizeRequest_BlankRepositoryFailsClosed pins D-16: a blank entry
// in the repository allow-set is rejected, never silently dropped.
func TestNormalizeRequest_BlankRepositoryFailsClosed(t *testing.T) {
	base := Request{Profile: "prod", Namespace: "view", Workspace: "ws", Project: "p"}

	blank := base
	blank.Repositories = []string{"repoA", " "}
	if _, err := NormalizeRequest(blank); err == nil {
		t.Fatal("NormalizeRequest with a blank repository entry succeeded, want an error")
	}

	duplicate := base
	duplicate.Repositories = []string{"repoA", "repoA"}
	got, err := NormalizeRequest(duplicate)
	if err != nil {
		t.Fatalf("NormalizeRequest: %v", err)
	}
	if want := []string{"repoA"}; !reflect.DeepEqual(got.Repositories, want) || !reflect.DeepEqual(got.Scope.Repositories, want) {
		t.Fatalf("Repositories = %#v, Scope.Repositories = %#v, want %#v", got.Repositories, got.Scope.Repositories, want)
	}
}
