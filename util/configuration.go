package util

type Configuration struct {
	Mappings        []Map            `mapstructure:"mappings"`
	MagentoBaseUrls []MagentoBaseUrl `mapstructure:"magento_base_urls"`
	Login           Login            `mapstructure:"login"`
	Sync            Sync             `mapstructure:"sync"`
	Context         ContextDefaults  `mapstructure:"context"`
	Behavior        Behavior         `mapstructure:"behavior"`
}

type ContextDefaults struct {
	OrgID    string `mapstructure:"org_id"`
	SiteID   string `mapstructure:"site_id"`
	ServerID string `mapstructure:"server_id"`
}

type Behavior struct {
	NonInteractive bool `mapstructure:"non_interactive"`
}

type Sync struct {
	Files SyncFiles `mapstructure:"files"`
}

type SyncFiles struct {
	Items []SyncFileItem `mapstructure:"items"`
}

type SyncFileItem struct {
	Source  string   `mapstructure:"source"`
	Target  string   `mapstructure:"target"`
	Exclude []string `mapstructure:"exclude"`
}

type Map struct {
	From string `mapstructure:"from"`
	To   string `mapstructure:"to"`
}

type MagentoBaseUrl struct {
	StoreCode string `mapstructure:"store_code"`
	RunType   string `mapstructure:"run_type"`
	Url       string
}

type Login struct {
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	Scope       string `mapstructure:"scope"`
	AuthMethod  string `mapstructure:"auth_method"`
	InstanceUrl string `mapstructure:"instance_url"`
}
