package s3fs

type Mount struct {
	ID     int64  `json:"id"`
	Path   string `json:"path"`
	Bucket string `json:"bucket"`
	URL    string `json:"url"`
}
