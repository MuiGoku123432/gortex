package neo4jprojection

import "testing"

// TestSensitiveMetadataKey_Phase4Keys proves every Meta key Phase 4 adds
// passes the secret-key screen, so the projection keeps it as a meta_
// property instead of dropping it.
func TestSensitiveMetadataKey_Phase4Keys(t *testing.T) {
	for _, key := range []string{
		"cobol_placeholder",
		"cobol_copy_library_key",
		"cobol_resource_raw",
		"cobol_copy_search_rank",
		"cobol_copy_search_rule",
		"cobol_qualifier_rule",
		"resolution_unresolved_reason",
		"resolution_candidate_ids",
	} {
		if sensitiveMetadataKey(key) {
			t.Errorf("Phase 4 key %q is classified sensitive and would be dropped", key)
		}
	}
}
