package main

// The ccl hook import below means building this will produce CCL'ed binaries.
// This file itself remains Apache2 to preserve the organization of ccl code
// under the /pkg/ccl subtree, but is unused for pure FLOSS builds.
import (
	_ "github.com/cockroachdb/cockroach/pkg/ccl" // ccl init hooks
	"github.com/cockroachdb/cockroach/pkg/cli"
)

func main() {
	cli.Main()
}
