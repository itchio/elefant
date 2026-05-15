package version

import (
	"strconv"
	"strings"
)

// GTOrEq reports whether v1 >= v2, comparing dot-separated numeric tokens
// left-to-right with missing trailing tokens treated as zero. Non-numeric
// tokens are also treated as zero.
func GTOrEq(v1 string, v2 string) bool {
	v1toks := strings.Split(v1, ".")
	v2toks := strings.Split(v2, ".")

	n := max(len(v1toks), len(v2toks))
	for i := range n {
		var n1, n2 int64
		if i < len(v1toks) {
			n1, _ = strconv.ParseInt(v1toks[i], 10, 64)
		}
		if i < len(v2toks) {
			n2, _ = strconv.ParseInt(v2toks[i], 10, 64)
		}
		if n1 != n2 {
			return n1 > n2
		}
	}
	return true
}
