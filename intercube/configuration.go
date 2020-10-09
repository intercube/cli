package intercube

type Configuration struct {
	Mappings []Map `mapstructure:"mappings"`
}

type Map struct {
	From string
	To   string
}
