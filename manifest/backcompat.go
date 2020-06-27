package manifest

import (
	"github.com/jtolds/jam/utils"
)

func (r *Range) Blob() string {
	if len(r.BlobBytes) > 0 {
		return utils.PathSafeIdEncode(r.BlobBytes)
	}
	return r.DeprecatedBlobString
}
