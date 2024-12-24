package confs

type ConfModel struct {
	ENV             string   `bind:"ENV"`
	Port            int      `bind:"PORT"`
	DomainWhitelist []string `bind:"DOMAIN_WHITELIST"`
	APIVersionName  string   `bind:"API_VERSION_NAME"`
	AuthKey         string   `bind:"AUTH_KEY"`
	AuthSecret      string   `bind:"AUTH_SECRET"`
}
