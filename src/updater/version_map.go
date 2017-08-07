package updater

import (
	"time"
)

// VersionMaps is an declaration for operations on a list of version maps
type VersionMaps []VersionMap

// VersionMap is the structure for mapping a UT4 build version to
// the symantic version number
type VersionMap struct {
	Version     string    `json:"version"`
	SemVer      string    `json:"semver"`
	ReleaseDate time.Time `json:"released"`
}

// GetVersionMapByVersionNumber retrieves the version map information
// based on the build version
func (versionMaps VersionMaps) GetVersionMapByVersionNumber(
	version string) VersionMap {
	for _, versionMap := range versionMaps {
		if versionMap.Version == version {
			return versionMap
		}
	}
	return VersionMap{}
}
