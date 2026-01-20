package alldebrid

type LinkUnlockResponse struct {
	Status string `json:"status"`
	Data   struct {
		Link     string `json:"link"`
		Filename string `json:"filename"`
		Filesize int64  `json:"filesize"`
	} `json:"data"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

type UserResponse struct {
	Status string `json:"status"`
	Data   struct {
		User struct {
			Username     string `json:"username"`
			IsPremium    bool   `json:"isPremium"`
			PremiumUntil int64  `json:"premiumUntil"`
		} `json:"user"`
	} `json:"data"`
}

type HostsResponse struct {
	Data struct {
		Hosts map[string]struct {
			Domain string `json:"domain"`
		} `json:"hosts"`
	} `json:"data"`
}
