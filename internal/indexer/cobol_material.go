package indexer

// cobolMaterialKey is the reserved RepoIndexState extractor-version key that
// carries the COBOL material fingerprint.
const cobolMaterialKey = "_cobol_material"

// cobolMaterial is a RED-phase stub.
func (idx *Indexer) cobolMaterial() (int64, bool) { return 0, false }
