package neo4jprojection

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/zzet/gortex/internal/graph"
)

var closedNeo4jToken = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

func projectionOwnerKey(namespace, workspace, project string, repositories []string) string {
	repos := append([]string(nil), repositories...)
	sort.Strings(repos)
	parts := []string{"owner", namespace, workspace, project}
	parts = append(parts, repos...)
	return projectionDigest("o", parts...)
}

func projectionNodeLogicalKey(owner, nodeID string) string {
	return projectionDigest("n", owner, nodeID)
}

func projectionEdgeLogicalKey(owner string, edge *graph.Edge) string {
	if edge == nil {
		return ""
	}
	occurrence := ""
	if edge.Meta != nil {
		for _, key := range []string{"occurrence", "occurrence_id", "discriminator"} {
			if value, ok := edge.Meta[key]; ok {
				occurrence = fmt.Sprintf("%T:%v", value, value)
				break
			}
		}
	}
	return projectionDigest("e", owner, edge.From, edge.To, string(edge.Kind), edge.FilePath, fmt.Sprint(edge.Line), edge.Origin, occurrence)
}

func projectionGenerationKey(owner, operation string, sourceGeneration int64) string {
	return projectionDigest("g", owner, operation, fmt.Sprint(sourceGeneration))
}

func projectionPhysicalKey(logical, generation string) string {
	return projectionDigest("p", logical, generation)
}

func projectionDigest(prefix string, values ...string) string {
	h := sha256.New()
	var size [8]byte
	for _, value := range values {
		binary.BigEndian.PutUint64(size[:], uint64(len(value)))
		_, _ = h.Write(size[:])
		_, _ = h.Write([]byte(value))
	}
	return prefix + "-" + hex.EncodeToString(h.Sum(nil)[:16])
}

func sanitizeLabel(kind graph.NodeKind) string {
	return sanitizeToken(string(kind), "Unknown", false)
}

func sanitizeRelationshipType(kind graph.EdgeKind) string {
	return sanitizeToken(string(kind), "RELATED", true)
}

func sanitizeToken(raw, fallback string, upper bool) string {
	if raw == "" {
		return fallback
	}
	var token strings.Builder
	for _, r := range raw {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			token.WriteRune(r)
		} else {
			token.WriteRune('_')
		}
	}
	base := strings.Trim(token.String(), "_")
	if base == "" {
		base = fallback
	}
	if base[0] >= '0' && base[0] <= '9' {
		base = "T_" + base
	}
	if upper {
		base = strings.ToUpper(base)
	} else {
		parts := strings.FieldsFunc(base, func(r rune) bool { return r == '_' })
		for i, part := range parts {
			if part != "" {
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
		base = strings.Join(parts, "")
	}
	if token.String() != raw || !closedNeo4jToken.MatchString(base) {
		sum := sha256.Sum256([]byte(raw))
		base += "_" + hex.EncodeToString(sum[:4])
	}
	return base
}
