package updater

// OSDistribution contains information about the OS and distribution
type OSDistribution struct {
	KernelVersion          string
	DistributionID         string
	Distribution           string
	DistributionVersion    string
	DistributionPrettyName string
}

// UpdateCheckRequest holds the information for update requests
type UpdateCheckRequest struct {
	ClientID       string         `json:"client_id"`
	OS             OSDistribution `json:"os"`
	Versions       []string       `json:"versions"`
	CurrentVersion string         `json:"current_version"`
}

// UpdateCheckResponse is the response for update check requests
type UpdateCheckResponse struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
}

// HashProgressEvent contains the progress event
type HashProgressEvent struct {
	Filename string
	Filepath string
	Error    string
	// MB/s processed
	Mbps float64
	// The estimated time to complete in seconds
	ETA       float64
	Percent   float64
	Completed bool
	Hash      string
}

// DownloadProgressEvent contains information about an ongoing download
type DownloadProgressEvent struct {
	Filename  string
	Mbps      float64
	ETA       float64
	Percent   float64
	Completed bool
}
