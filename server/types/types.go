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
}

/*
LogReq: where

	AppId = app ID
	Priority = low watermark priority level
	Age = number of days to cast net backwards
*/
type LogReq struct {
	AppId    string `json:"app"`
	Priority string `json:"pri"`
	Days     int64  `json:"days"`
}
