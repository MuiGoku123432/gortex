package neo4jprojection

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/zzet/gortex/internal/graph"
)

const (
	maxMetadataDepth    = 8
	maxMetadataElements = 10_000
	maxMetadataBytes    = 1 << 20
)

type projectedNode struct {
	Labels     []string       `json:"labels"`
	Properties map[string]any `json:"properties"`
}

type projectedEdge struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

type projectionWarnings struct {
	Secret      int `json:"secret"`
	Unsupported int `json:"unsupported"`
}

func projectNode(owner, generation string, node *graph.Node) (projectedNode, projectionWarnings, error) {
	if node == nil || owner == "" || generation == "" || node.ID == "" {
		return projectedNode{}, projectionWarnings{}, fmt.Errorf("projection node identity, owner, and generation are required")
	}
	logical := projectionNodeLogicalKey(owner, node.ID)
	properties := map[string]any{
		"gortex_owner": owner, "gortex_generation": generation,
		"gortex_logical_key": logical, "gortex_physical_key": projectionPhysicalKey(logical, generation),
		"id": node.ID, "kind": string(node.Kind), "name": node.Name, "qual_name": node.QualName,
		"file_path": node.FilePath, "start_line": int64(node.StartLine), "end_line": int64(node.EndLine),
		"start_column": int64(node.StartColumn), "end_column": int64(node.EndColumn), "language": node.Language,
		"repo_prefix": node.RepoPrefix, "workspace_id": node.WorkspaceID, "project_id": node.ProjectID,
		"origin": node.Origin, "stub": node.Stub, "fetched_at": node.FetchedAt.UTC().Format(time.RFC3339Nano),
	}
	warnings := appendMetadata(properties, node.Meta)
	labels := []string{"GortexNode", sanitizeLabel(node.Kind)}
	if graph.IsUnresolvedTarget(node.ID) || strings.HasPrefix(node.ID, "external::") || string(node.Kind) == "unresolved" || node.Meta != nil && node.Meta["synthetic"] == true {
		labels = append(labels, "GortexUnresolved")
	}
	return projectedNode{Labels: labels, Properties: properties}, warnings, nil
}

func projectEdge(owner, generation string, edge *graph.Edge) (projectedEdge, projectionWarnings, error) {
	if edge == nil || owner == "" || generation == "" || edge.From == "" || edge.To == "" || edge.Kind == "" {
		return projectedEdge{}, projectionWarnings{}, fmt.Errorf("projection edge identity, owner, generation, endpoints, and kind are required")
	}
	logical := projectionEdgeLogicalKey(owner, edge)
	properties := map[string]any{
		"gortex_owner": owner, "gortex_generation": generation,
		"gortex_logical_key": logical, "gortex_physical_key": projectionPhysicalKey(logical, generation),
		"from": edge.From, "to": edge.To, "kind": string(edge.Kind), "file_path": edge.FilePath,
		"line": int64(edge.Line), "confidence": edge.Confidence, "confidence_label": edge.ConfidenceLabel,
		"origin": edge.Origin, "tier": edge.Tier, "cross_repo": edge.CrossRepo, "context": edge.Context,
		"return_usage": edge.ReturnUsage, "via": edge.Via, "alias": edge.Alias, "name_only": edge.NameOnly,
	}
	warnings := appendMetadata(properties, edge.Meta)
	return projectedEdge{Type: sanitizeRelationshipType(edge.Kind), Properties: properties}, warnings, nil
}

func appendMetadata(properties map[string]any, metadata map[string]any) projectionWarnings {
	var warnings projectionWarnings
	if len(metadata) == 0 {
		return warnings
	}
	budget := metadataBudget{}
	keys := make([]string, 0, min(len(metadata), maxMetadataElements))
	for key := range metadata {
		if sensitiveMetadataKey(key) {
			warnings.Secret++
			continue
		}
		// Charge each source key before retaining it for sorting. The factor
		// bounds the original key, escaped property name, key-map entry, and
		// JSON quoting/separator allocation before any of those are built.
		budget.elements++
		budget.bytes += len(key)*4 + len("meta_\"\":\"\",")
		if budget.elements > maxMetadataElements || budget.bytes > maxMetadataBytes {
			warnings.Unsupported += len(metadata) - warnings.Secret - len(keys)
			return warnings
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	keyMap := make(map[string]string)
	encodings := make(map[string]string)
	used := make(map[string]string)
	for _, key := range keys {
		encoded := encodeMetadataKey(key)
		if prior, exists := used[encoded]; exists && prior != key {
			encoded += "_" + projectionDigest("", key)[1:9]
		}
		used[encoded] = key
		propertyKey := "meta_" + encoded
		before := budget
		value, encoding, ok := projectMetadataValueWithBudget(metadata[key], &budget)
		if !ok {
			warnings.Unsupported++
			continue
		}
		// Charge the generated encoding-map entry only when present. Key-map
		// space was conservatively charged during the allocation-safe preflight.
		if encoding != "" {
			budget.elements++
			budget.bytes += len(propertyKey) + len(encoding) + len("\"\":\"\",")
		}
		if budget.elements > maxMetadataElements || budget.bytes > maxMetadataBytes {
			budget = before
			warnings.Unsupported++
			continue
		}
		properties[propertyKey] = value
		keyMap[propertyKey] = key
		if encoding != "" {
			encodings[propertyKey] = encoding
		}
	}
	if len(keyMap) > 0 {
		properties["gortex_meta_key_map"] = mustCanonicalJSONString(keyMap)
	}
	if len(encodings) > 0 {
		properties["gortex_meta_encoding"] = mustCanonicalJSONString(encodings)
	}
	return warnings
}

func encodeMetadataKey(key string) string {
	if key == "" {
		return "x00"
	}
	var out strings.Builder
	for i := 0; i < len(key); i++ {
		b := key[i]
		if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' {
			out.WriteByte(b)
		} else {
			out.WriteString("x")
			out.WriteString(hex.EncodeToString([]byte{b}))
		}
	}
	return out.String()
}

func sensitiveMetadataKey(key string) bool {
	normalized := strings.ToLower(key)
	normalized = strings.NewReplacer("-", "", "_", "", ".", "", " ", "").Replace(normalized)
	for _, secret := range []string{"password", "passwd", "secret", "token", "credential", "privatekey", "apikey", "authorization"} {
		if strings.Contains(normalized, secret) {
			return true
		}
	}
	return false
}

type metadataBudget struct {
	elements int
	bytes    int
}

func projectMetadataValue(value any) (any, string, bool) {
	return projectMetadataValueWithBudget(value, &metadataBudget{})
}

func projectMetadataValueWithBudget(value any, budget *metadataBudget) (any, string, bool) {
	before := *budget
	if !budget.preflight(reflect.ValueOf(value), 0) {
		*budget = before
		return nil, "", false
	}
	projected, encoding, ok := projectMetadataValueAtDepth(value, 0)
	if !ok {
		*budget = before
	}
	return projected, encoding, ok
}

func (b *metadataBudget) preflight(value reflect.Value, depth int) bool {
	if !value.IsValid() || depth > maxMetadataDepth {
		return false
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return false
		}
		value = value.Elem()
	}
	b.elements++
	if b.elements > maxMetadataElements {
		return false
	}
	switch value.Kind() {
	case reflect.String:
		b.bytes += value.Len()
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		b.bytes += int(value.Type().Size())
	case reflect.Slice, reflect.Array:
		if value.Len() > maxMetadataElements-b.elements {
			return false
		}
		for i := 0; i < value.Len(); i++ {
			if !b.preflight(value.Index(i), depth+1) {
				return false
			}
		}
	case reflect.Map:
		if value.Len() > maxMetadataElements-b.elements {
			return false
		}
		iter := value.MapRange()
		for iter.Next() {
			if !b.preflight(iter.Key(), depth+1) || !b.preflight(iter.Value(), depth+1) {
				return false
			}
		}
	case reflect.Struct:
		if value.Type() == reflect.TypeOf(time.Time{}) {
			b.bytes += len(time.RFC3339Nano)
			break
		}
		for i := 0; i < value.NumField(); i++ {
			if value.Type().Field(i).IsExported() && !b.preflight(value.Field(i), depth+1) {
				return false
			}
		}
	default:
		return false
	}
	return b.bytes <= maxMetadataBytes
}

func projectMetadataValueAtDepth(value any, depth int) (any, string, bool) {
	if depth > maxMetadataDepth {
		return nil, "", false
	}
	if value == nil {
		return nil, "", false
	}
	switch typed := value.(type) {
	case string, bool, int64:
		return typed, "", true
	case int:
		return int64(typed), "", true
	case int8:
		return int64(typed), "", true
	case int16:
		return int64(typed), "", true
	case int32:
		return int64(typed), "", true
	case uint:
		if uint64(typed) > math.MaxInt64 {
			return nil, "", false
		}
		return int64(typed), "", true
	case uint8:
		return int64(typed), "", true
	case uint16:
		return int64(typed), "", true
	case uint32:
		return int64(typed), "", true
	case uint64:
		if typed > math.MaxInt64 {
			return nil, "", false
		}
		return int64(typed), "", true
	case float32:
		v := float64(typed)
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, "", false
		}
		return v, "", true
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return nil, "", false
		}
		return typed, "", true
	case time.Time:
		return typed.UTC().Format(time.RFC3339Nano), "time", true
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		out := make([]any, rv.Len())
		var elementType reflect.Type
		for i := 0; i < rv.Len(); i++ {
			projected, encoding, ok := projectMetadataValueAtDepth(rv.Index(i).Interface(), depth+1)
			if !ok || encoding != "" || elementType != nil && reflect.TypeOf(projected) != elementType {
				data, err := canonicalJSON(value)
				return string(data), "json", err == nil && len(data) <= maxMetadataBytes
			}
			out[i] = projected
			elementType = reflect.TypeOf(projected)
		}
		return out, "", true
	}
	if rv.Kind() == reflect.Map || rv.Kind() == reflect.Struct {
		data, err := canonicalJSON(value)
		return string(data), "json", err == nil && len(data) <= maxMetadataBytes
	}
	return nil, "", false
}

func canonicalJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, data); err != nil {
		return nil, err
	}
	return compact.Bytes(), nil
}

func mustCanonicalJSONString(value any) string {
	data, err := canonicalJSON(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}
