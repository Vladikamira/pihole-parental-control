package pihole

type AuthResponse struct {
	Session struct {
		Valid    bool   `json:"valid"`
		Totp     bool   `json:"totp"`
		Sid      string `json:"sid"`
		Csrf     string `json:"csrf"`
		Validity int    `json:"validity"`
		Message  string `json:"message"`
	} `json:"session"`
	Took float64 `json:"took"`
}

type QueryStats struct {
	Queries         []Query `json:"queries"`
	Cursor          int     `json:"cursor"`
	RecordsTotal    int     `json:"recordsTotal"`
	RecordsFiltered int     `json:"recordsFiltered"`
	Draw            int     `json:"draw"`
	Took            float64 `json:"took"`
}

type Query struct {
	ID       int         `json:"id"`
	Time     float64     `json:"time"`
	Type     string      `json:"type"`
	Status   string      `json:"status"`
	DNSSEC   string      `json:"dnssec"`
	Domain   string      `json:"domain"`
	Upstream *string     `json:"upstream"`
	Reply    QueryReply  `json:"reply"`
	Client   QueryClient `json:"client"`
	ListID   *int        `json:"list_id"`
	EDE      QueryEDE    `json:"ede"`
	CNAME    *string     `json:"cname"`
}

type QueryReply struct {
	Type string  `json:"type"`
	Time float64 `json:"time"`
}

type QueryClient struct {
	IP   string  `json:"ip"`
	Name *string `json:"name"`
}

type QueryEDE struct {
	Code int     `json:"code"`
	Text *string `json:"text"`
}

type Group struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Comment      string `json:"comment"`
	Enabled      bool   `json:"enabled"`
	DateAdded    int    `json:"date_added"`
	DateModified int    `json:"date_modified"`
}

type GroupResponse struct {
	Groups []Group `json:"groups"`
	Took   float64 `json:"took"	`
}

type GroupListResponse struct {
	Groups []Group `json:"groups"`
}

type ClientItem struct {
	IP      string `json:"client"`
	Comment string `json:"comment"`
	Groups  []int  `json:"groups"`
}

type DomainItem struct {
	ID      int    `json:"id"`
	Domain  string `json:"domain"`
	Type    int    `json:"type"` // 0: whitelist exact, 1: blacklist exact, 2: whitelist regex, 3: blacklist regex
	Comment string `json:"comment"`
	Groups  []int  `json:"groups"`
}
