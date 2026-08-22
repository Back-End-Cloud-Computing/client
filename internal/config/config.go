// Package config lê a configuração do ambiente.
//
// Nada é lido de arquivo de configuração versionado: o mesmo binário roda em
// desenvolvimento, teste e produção trocando só as variáveis.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	ServerPort       string
	DatabaseURL      string
	JWTPublicKeyPath string
	JWTIssuer        string
	AllowedOrigins   []string
}

func Load() (Config, error) {
	cfg := Config{
		ServerPort:       env("SERVER_PORT", "8082"),
		JWTPublicKeyPath: env("JWT_PUBLIC_KEY_PATH", "keys/public.pem"),
		JWTIssuer:        env("JWT_ISSUER", "ganjj-authorization"),
		AllowedOrigins: strings.Split(env("CORS_ALLOWED_ORIGINS",
			"http://localhost:4200,http://localhost:5173,http://localhost:3000"), ","),
	}

	// As mesmas variáveis usadas pelos outros serviços do GANJJ.
	host := env("DB_HOST", "localhost")
	porta := env("DB_PORT", "5432")
	nome := env("DB_NAME", "ganjj_client")
	usuario := env("DB_USER", "ganjj_client")
	senha := env("DB_PASSWORD", "ganjj_client")

	cfg.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		url.QueryEscape(usuario), url.QueryEscape(senha), host, porta, nome,
		env("DB_SSLMODE", "disable"))

	for i := range cfg.AllowedOrigins {
		cfg.AllowedOrigins[i] = strings.TrimSpace(cfg.AllowedOrigins[i])
	}

	return cfg, nil
}

func env(chave, padrao string) string {
	if valor := os.Getenv(chave); valor != "" {
		return valor
	}
	return padrao
}
