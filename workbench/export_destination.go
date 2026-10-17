package workbench

import (
	"os"
	"path/filepath"
)

// Export must not replace the associated schematic through a different spelling
// of its directory or a file alias. These filesystem observations belong in the
// worker and are repeated before publication; unrelated IO errors retain the
// ordinary preparation checks. This is not filesystem compare-and-swap.
func checkExportDestination(path, protectedPath string) error {
	if protectedPath == "" {
		return nil
	}
	refuse := &actionError{"invalid-params", "export must use a different file from the associated schematic, including filesystem aliases"}
	if path == protectedPath {
		return refuse
	}
	// Resolve directories separately so the protected name remains protected
	// even if an outside process has removed the associated file.
	destinationDir, destinationErr := filepath.EvalSymlinks(filepath.Dir(path))
	protectedDir, protectedErr := filepath.EvalSymlinks(filepath.Dir(protectedPath))
	if destinationErr == nil && protectedErr == nil && filepath.Join(destinationDir, filepath.Base(path)) == filepath.Join(protectedDir, filepath.Base(protectedPath)) {
		return refuse
	}
	destination, destinationErr := os.Stat(path)
	protected, protectedErr := os.Stat(protectedPath)
	if destinationErr == nil && protectedErr == nil && os.SameFile(destination, protected) {
		return refuse
	}
	return nil
}
