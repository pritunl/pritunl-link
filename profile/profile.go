package profile

var (
	Host         string
	Token        string
	Secret       string
	Username     string
	NetworkLinks []string
)

type Profile struct {
	UserId         string   `json:"user_id"`
	OrganizationId string   `json:"organization_id"`
	ServerId       string   `json:"server_id"`
	SyncHash       string   `json:"sync_hash"`
	SyncToken      string   `json:"sync_token"`
	SyncSecret     string   `json:"sync_secret"`
	SyncHosts      []string `json:"sync_hosts"`
	Conf           string   `json:"conf"`
}
