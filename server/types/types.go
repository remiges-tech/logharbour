package types



type AppConfig struct {
	DBConnURL              string `json:"db_conn_url"`
	DBHost                 string `json:"db_host"`
	DBPort                 int    `json:"db_port"`
	DBUser                 string `json:"db_user"`
	DBPassword             string `json:"db_password"`
	IndexName              string `json:"index_name"`
	AppServerPort          string `json:"app_server_port"`
	ProviderUrl            string `json:"provider_url"`
	KeycloakURL            string `json:"keycloak_url"`
	KeycloakClientID       string `json:"keycloak_client_id"`
	CertificateFingerprint string `json:"certificate_fingerprint"`
	GeoLiteDbPath          string  `json:"geolite_db_path"`
}


