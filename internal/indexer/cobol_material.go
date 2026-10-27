package indexer

import (
	"encoding/binary"
	"encoding/json"
	"strconv"

	"github.com/zeebo/blake3"

	"github.com/zzet/gortex/internal/config"
)

// cobolMaterialKey is the reserved RepoIndexState extractor-version key that
// carries the COBOL material fingerprint. Like the post-extraction policy
// key it is never returned as a language to re-stage.
const cobolMaterialKey = "_cobol_material"

// cobolMaterialFingerprint digests what COBOL observations are derived from
// besides their bytes (D-06): the extractor's stamped identity and this
// repository's index.cobol block (allowlist D-09, library order D-10,
// default qualifiers D-11). It is the first 6 bytes of the BLAKE3 digest as
// a big-endian integer, so it fits the integer map RepoIndexState persists.
//
// PURPOSE — prov_extractor_version was never compared on reindex and the
// whole-config hash is too coarse, so an index.cobol edit with unchanged
// source bytes left stale bindings in place, and an unrelated key would
// restage every COBOL file if the whole config were hashed.
//
// RATIONALE — encoding/json writes struct fields in declaration order, so
// the encoding of cfg is stable. There is no retrieval manifest input yet:
// the revision that supplied the bytes is handled by the VCS re-stamp
// (D-17), and a manifest-driven search order is a deferred idea. A 48-bit
// collision would skip one restage; it is only a change detector.
//
// KEYWORDS — cobol, material, fingerprint, staleness, restage, index.cobol
func cobolMaterialFingerprint(identity string, cfg config.CobolConfig) int64 {
	// A struct of strings and string slices always encodes.
	encoded, _ := json.Marshal(cfg)
	sum := blake3.Sum256([]byte(identity + "\n" + string(encoded)))
	var buf [8]byte
	copy(buf[2:], sum[:6])
	return int64(binary.BigEndian.Uint64(buf[:]))
}

// cobolMaterial returns this repository's COBOL material fingerprint. ok is
// false when the registry has no COBOL extractor that reports its material
// identity (the non-darwin/linux stub), so no fingerprint is persisted or
// compared.
func (idx *Indexer) cobolMaterial() (int64, bool) {
	if idx.registry == nil {
		return 0, false
	}
	ext, _ := idx.registry.GetByLanguage("cobol")
	m, ok := ext.(interface{ MaterialIdentity() string })
	if !ok {
		return 0, false
	}
	return cobolMaterialFingerprint(m.MaterialIdentity(), idx.cobolConfig()), true
}

// repoExtractorVersionsSnapshot is extractorVersionsSnapshot plus this
// repository's COBOL material fingerprint under cobolMaterialKey, when one
// is available. It is what persistRepoIndexState records and what the
// repository's own staleness check compares against.
func (idx *Indexer) repoExtractorVersionsSnapshot() map[string]int {
	out := extractorVersionsSnapshot()
	if fp, ok := idx.cobolMaterial(); ok {
		out[cobolMaterialKey] = int(fp)
	}
	return out
}

// merkleSaltFor is the Merkle leaf salt for one repository-relative path:
// the package-level merkleSaltFor plus, for a COBOL file, this repository's
// material fingerprint (D-06).
//
// PURPOSE — Merkle mode never consults the stored _cobol_material, so
// without this an index.cobol edit or an extractor or parser identity change
// with unchanged bytes would keep serving stale COBOL observations there.
//
// RATIONALE — the salt is part of the leaf, so a fingerprint change re-flags
// exactly the COBOL leaves without re-reading them, and the next pass saves
// the new salt, so the restage happens once. Other languages' salts are
// unchanged.
//
// KEYWORDS — merkle, salt, cobol, material, fingerprint, restage
func (idx *Indexer) merkleSaltFor(rel string) string {
	salt := merkleSaltFor(rel)
	if ExtractorLangForFile(rel) != "cobol" {
		return salt
	}
	fp, ok := idx.cobolMaterial()
	if !ok {
		return salt
	}
	return salt + "|cobol-material@" + strconv.FormatInt(fp, 10)
}
