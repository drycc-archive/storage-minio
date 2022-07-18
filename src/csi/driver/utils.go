package driver

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"strings"
)

func sanitizeVolumeID(volumeID string) string {
	volumeID = strings.ToLower(volumeID)
	if len(volumeID) > 63 {
		h := sha1.New()
		io.WriteString(h, volumeID)
		volumeID = hex.EncodeToString(h.Sum(nil))
	}
	return volumeID
}

// volumeIDToBucketPrefix returns the bucket name and prefix based on the volumeID.
// Prefix is empty if volumeID does not have a slash in the name.
func volumeIDToBucketPrefix(volumeID string) (string, string) {
	// if the volumeID has a slash in it, this volume is
	// stored under a certain prefix within the bucket.
	splitVolumeID := strings.SplitN(volumeID, "/", 2)
	if len(splitVolumeID) > 1 {
		return splitVolumeID[0], splitVolumeID[1]
	}

	return volumeID, ""
}
