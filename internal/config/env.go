package config

type Env struct {
	JobNo     string
	One       OneConfig
	Endpoints EndpointConfig
}

type OneConfig struct {
	AuthToken string
}

type EndpointConfig struct {
	LeaveListURL   string
	LeaveCheckURL  string
	LeaveCreateURL string
}
