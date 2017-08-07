package updater

// Version holds information about an installed UT4 version
type Version struct {
	VersionMap
	Path string
}

// ByVersion allows for sorting by the build version number
type ByVersion []Version

func (a ByVersion) Len() int           { return len(a) }
func (a ByVersion) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByVersion) Less(i, j int) bool { return a[i].Version > a[j].Version }
