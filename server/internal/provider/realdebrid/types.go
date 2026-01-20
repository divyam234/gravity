package realdebrid

type UnrestrictLinkResponse struct {
	Link     string `json:"link"`
	Filename string `json:"filename"`
	Filesize int64  `json:"filesize"`
	Error    string `json:"error"`
}

type UserResponse struct {
	Username   string `json:"username"`
	Type       string `json:"type"` // "premium"
	Expiration string `json:"expiration"`
}
