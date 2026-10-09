package platform

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL, Addr, APIURL, Issuer, SiteURL, DataDir string
	Origins                                             []string
	Production, Registration                            bool
	SMTPAddr, SMTPUser, SMTPPassword, MailFrom          string
	GoogleID, GoogleSecret, GitHubID, GitHubSecret      string
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
func LoadConfig() (Config, error) {
	c := Config{DatabaseURL: os.Getenv("DATABASE_URL"), Addr: env("LISTEN_ADDR", ":8080"), APIURL: env("API_URL", "http://localhost:8080"), Issuer: env("ISSUER_URL", "http://localhost:8080"), SiteURL: env("SITE_URL", "http://localhost:3000"), DataDir: env("DATA_DIR", "./data"), Production: os.Getenv("APP_ENV") == "production", Registration: os.Getenv("REGISTRATION_ENABLED") == "true", SMTPAddr: env("SMTP_ADDR", "mail:587"), SMTPUser: os.Getenv("SMTP_USER"), SMTPPassword: os.Getenv("SMTP_PASSWORD"), MailFrom: os.Getenv("MAIL_FROM"), GoogleID: os.Getenv("GOOGLE_CLIENT_ID"), GoogleSecret: os.Getenv("GOOGLE_CLIENT_SECRET"), GitHubID: os.Getenv("GITHUB_CLIENT_ID"), GitHubSecret: os.Getenv("GITHUB_CLIENT_SECRET")}
	c.Origins = strings.Split(env("FRONTEND_ORIGINS", c.SiteURL), ",")
	for _, v := range append(append([]string{}, c.Origins...), c.APIURL, c.Issuer, c.SiteURL) {
		u, e := url.Parse(v)
		if e != nil || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil || (c.Production && u.Scheme != "https") {
			return c, fmt.Errorf("invalid origin %q", v)
		}
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	if c.Registration && (c.SMTPUser == "" || c.SMTPPassword == "" || c.MailFrom == "") {
		return c, fmt.Errorf("registration requires authenticated mail configuration")
	}
	return c, nil
}
