package intercube

type Configuration struct {
	Mappings        []Map            `mapstructure:"mappings"`
	MagentoBaseUrls []MagentoBaseUrl `mapstructure:"magento_base_urls"`
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
