package main

type Configuration struct {
	Mappings        []Map            `mapstructure:"mappings"`
	MagentoBaseUrls []MagentoBaseUrl `mapstructure:"magento_base_urls"`
	Login           Login            `mapstructure:"login"`
}

type Map struct {
	From string
	To   string
}

type MagentoBaseUrl struct {
	StoreCode string `mapstructure:"store_code"`
	RunType   string `mapstructure:"run_type"`
	Url       string
}

type Login struct {
	Username   string
	Password   string
	Scope      string
	AuthMethod string `mapstructure:"auth_method"`
}
