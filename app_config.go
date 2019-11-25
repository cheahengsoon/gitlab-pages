package main

type appConfig struct {
	Domain                 string
	ArtifactsServer        string
	ArtifactsServerTimeout int
	RootCertificate        []byte
	RootKey                []byte
	MaxConns               int

	ListenHTTP      []uintptr
	ListenHTTPS     []uintptr
	ListenProxy     []uintptr
	ListenMetrics   uintptr
	InsecureCiphers bool
	TLSMinVersion   uint16
	TLSMaxVersion   uint16

	HTTP2        bool
	RedirectHTTP bool
	StatusPath   string

	DisableCrossOriginRequests bool

	LogFormat  string
	LogVerbose bool

	StoreSecret       string
	GitLabServer      string
	ClientID          string
	ClientSecret      string
	RedirectURI       string
	SentryDSN         string
	SentryEnvironment string
	CustomHeaders     []string
}

// GitlabServerURL returns URL to a GitLab instance.
func (config appConfig) GitlabServerURL() string {
	return config.GitLabServer
}

// GitlabClientSecret returns GitLab server access token.
func (config appConfig) GitlabClientSecret() []byte {
	return []byte(config.ClientSecret)
}
