# Gonum numerical dependency

ELab pins `gonum.org/v1/gonum v0.17.0` in the root Go module and checksums in
`go.sum`; the desktop's transitive requirement is pinned as well. Go's normal
module checksum verification remains enabled. No SDK or sibling checkout is
modified. The pure-Go `mat.Eigen` implementation supplies the checked local
storage-mode decomposition; ELab does not introduce an ad hoc eigen solver.

Upstream: https://github.com/gonum/gonum/releases/tag/v0.17.0
API: https://pkg.go.dev/gonum.org/v1/gonum@v0.17.0/mat#Eigen
License: [BSD-3-Clause](LICENSE). Include this notice with every distributed
binary incorporating Gonum. Candidate packages must copy it into Licenses and
include it in any binary-bound app notice bundle passed to the pinned packager.
This dependency does not require runtime source/module access.
