package payload

// PictureResponse mirrors the legacy PictureVo: an object in S3 plus the metadata
// the gallery needs to render it without another request.
type PictureResponse struct {
	ID               int64         `json:"id"`
	Filename         string        `json:"filename"`
	OriginalFilename string        `json:"original_filename"`
	MimeType         string        `json:"mime_type"`
	Size             int64         `json:"size"`
	URL              string        `json:"url"`
	ThumbnailURL     string        `json:"thumbnail_url"`
	Alt              string        `json:"alt"`
	ShowInGallery    bool          `json:"show_in_gallery"`
	Tags             []TagResponse `json:"tags"`
	CreatedAt        string        `json:"created_at"`
}
