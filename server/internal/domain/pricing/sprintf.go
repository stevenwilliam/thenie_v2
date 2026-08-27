package pricing

import "fmt"

// sprintf exists only so tier.go reads without fmt noise in the branch ladder,
// where the shape of the ladder is the thing worth seeing.
func sprintf(format string, args ...any) string { return fmt.Sprintf(format, args...) }
