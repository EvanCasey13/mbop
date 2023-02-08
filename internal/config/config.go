package config

import "os"

type MbopConfig struct {
	MailerModule           string
	JwtModule              string
	JwkURL                 string
	UsersModule            string
	CognitoAppClientID     string
	CognitoAppClientSecret string
	CognitoScope           string
	OauthTokenURL          string
	AmsURL                 string

	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
}

var conf *MbopConfig

func Get() *MbopConfig {
	if conf != nil {
		return conf
	}

	c := &MbopConfig{
		UsersModule:  fetchWithDefault("USERS_MODULE", ""),
		JwtModule:    fetchWithDefault("JWT_MODULE", ""),
		JwkURL:       fetchWithDefault("JWK_URL", ""),
		MailerModule: fetchWithDefault("MAILER_MODULE", "print"),

		DatabaseHost:     fetchWithDefault("DATABASE_HOST", "localhost"),
		DatabasePort:     fetchWithDefault("DATABASE_PORT", "5432"),
		DatabaseUser:     fetchWithDefault("DATABASE_USER", "postgres"),
		DatabasePassword: fetchWithDefault("DATABASE_PASSWORD", ""),
		DatabaseName:     fetchWithDefault("DATABASE_NAME", "mbop"),

		CognitoAppClientID:     fetchWithDefault("COGNITO_APP_CLIENT_ID", ""),
		CognitoAppClientSecret: fetchWithDefault("COGNITO_APP_CLIENT_SECRET", ""),
		CognitoScope:           fetchWithDefault("COGNITO_SCOPE", ""),
		OauthTokenURL:          fetchWithDefault("OAUTH_TOKEN_URL", ""),
		AmsURL:                 fetchWithDefault("AMS_URL", ""),
	}

	conf = c
	return conf
}

func fetchWithDefault(name, defaultValue string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}

	return defaultValue
}

// TO BE USED FROM TESTING ONLY.
func Reset() {
	conf = nil
}
