package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// server modes
const (
	DEVELOP_MODE    = "develop"
	PRODUCTION_MODE = "production"
	TEST_MODE       = "test"
)

// config holds all configuration for the server. It pulls from three places (in order):
// 		1. environment variables
// 		2. config.[server_mode].json <- eg: config.test.json
// 		3. config.json
//
// env variables win, but can only set config who's json is ALL_CAPS
// it's totally fine to not have, say, config.develop.json defined, and just
// rely on a base config.json. But if you're in production mode & config.production.json
// exists, that will be read *instead* of config.json.
//
// configuration is read at startup and cannot be alterd without restarting the server.
type config struct {
	// path to go source code
	Gopath string `json:"GOPATH"`

	// port to listen on, will be read from PORT env variable if present.
	Port string `json:"PORT"`

	// root url for service
	UrlRoot string `json:"URL_ROOT"`

	// url of postgres app db
	PostgresDbUrl string `json:"POSTGRES_DB_URL"`

	// Public Key to use for signing metablocks. required.
	PublicKey string `json:"PUBLIC_KEY"`

	// TLS (HTTPS) enable support via LetsEncrypt, default false
	// should be true in production
	TLS bool `json:"TLS"`

	// How long before a url is considered stale, in hours.
	StaleDurationHours time.Duration `json:"stale_duration_hours"`

	// Content Types to Store
	StoreContentTypes []string

	// read from env variable: AWS_REGION
	// the region your bucket is in, eg "us-east-1"
	AwsRegion string `json:"AWS_REGION"`
	// read from env variable: AWS_S3_BUCKET_NAME
	// should be just the name of your bucket, no protocol prefixes or paths
	AwsS3BucketName string `json:"AWS_S3_BUCKET_NAME"`
	// read from env variable: AWS_ACCESS_KEY_ID
	AwsAccessKeyId string `json:"AWS_ACCESS_KEY_ID"`
	// read from env variable: AWS_SECRET_ACCESS_KEY
	AwsSecretAccessKey string `json:"AWS_SECRET_ACCESS_KEY"`
	// path to store & retrieve data from
	AwsS3BucketPath string `json:"AWS_S3_BUCKET_PATH"`

	// setting HTTP_AUTH_USERNAME & HTTP_AUTH_PASSWORD
	// will enable basic http auth for the server. This is a single
	// username & password that must be passed in with every request.
	// leaving these values blank will disable http auth
	// read from env variable: HTTP_AUTH_USERNAME
	HttpAuthUsername string `json:"HTTP_AUTH_USERNAME"`
	// read from env variable: HTTP_AUTH_PASSWORD
	HttpAuthPassword string `json:"HTTP_AUTH_PASSWORD"`

	// if true, requests that have X-Forwarded-Proto: http will be redirected
	// to their https variant
	ProxyForceHttps bool

	// config used for rendering to templates. in config.json set
	// template_data to an object, and anything provided there
	// will be available to the templates in the views directory.
	// index.html has an example of using template_data to set the "title"
	// attribute
	TemplateData map[string]interface{} `json:"template_data"`

	// CertbotResponse is only for doing manual SSL certificate generation
	// via LetsEncrypt.
	CertbotResponse string `json:"CERTBOT_RESPONSE"`
}

// StaleDuration turns cfg.StaleDurationHours into a time.Duration
func (cfg *config) StaleDuration() time.Duration {
	return cfg.StaleDurationHours * time.Hour
}

// initConfig pulls configuration from config.json
func initConfig(mode string) (cfg *config, err error) {
	cfg = &config{}

	if err := loadConfigFile(mode, cfg); err != nil {
		return cfg, err
	}

	// override config settings with env settings, passing in the current configuration
	// as the default. This has the effect of leaving the config.json value unchanged
	// if the env variable is empty
	cfg.Gopath = readEnvString("GOPATH", cfg.Gopath)
	cfg.Port = readEnvString("PORT", cfg.Port)
	cfg.UrlRoot = readEnvString("URL_ROOT", cfg.UrlRoot)
	cfg.PublicKey = readEnvString("PUBLIC_KEY", cfg.PublicKey)
	cfg.TLS = readEnvBool("TLS", cfg.TLS)
	cfg.PostgresDbUrl = readEnvString("POSTGRES_DB_URL", cfg.PostgresDbUrl)
	cfg.HttpAuthUsername = readEnvString("HTTP_AUTH_USERNAME", cfg.HttpAuthUsername)
	cfg.HttpAuthPassword = readEnvString("HTTP_AUTH_PASSWORD", cfg.HttpAuthPassword)
	cfg.AwsRegion = readEnvString("AWS_REGION", cfg.AwsRegion)
	cfg.AwsS3BucketName = readEnvString("AWS_S3_BUCKET_NAME", cfg.AwsS3BucketName)
	cfg.AwsS3BucketPath = readEnvString("AWS_S3_BUCKET_PATH", cfg.AwsS3BucketPath)
	cfg.AwsAccessKeyId = readEnvString("AWS_ACCESS_KEY_ID", cfg.AwsAccessKeyId)
	cfg.AwsSecretAccessKey = readEnvString("AWS_SECRET_ACCESS_KEY", cfg.AwsSecretAccessKey)
	cfg.CertbotResponse = readEnvString("CERTBOT_RESPONSE", cfg.CertbotResponse)
	// cfg.StaleDuration = readEnvInt("STALE_DURATION", cfg.StaleDuration)

	// make sure port is set
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	err = requireConfigStrings(map[string]string{
		"GOPATH":          cfg.Gopath,
		"PORT":            cfg.Port,
		"POSTGRES_DB_URL": cfg.PostgresDbUrl,
		"PUBLIC_KEY":      cfg.PublicKey,
	})

	if cfg.TemplateData == nil {
		// TODO - this shouldn't be here. inferring defaults for dockerized dev
		cfg.TemplateData = map[string]interface{}{
			"title":           "Archivers",
			"segmentApiToken": "",
			"webappScripts": []string{
				"http://localhost:4000/static/bundle.js",
			},
		}
	}

	if cfg.TemplateData["ENV"] == nil {
		cfg.TemplateData["ENV"] = mode
	}

	templates = template.Must(template.ParseFiles(
		packagePath("views/webapp.html"),
		packagePath("views/accessDenied.html"),
		packagePath("views/notFound.html"),
	))

	return
}

func packagePath(path string) string {
	return filepath.Join(os.Getenv("GOPATH"), "src/github.com/archivers-space/patchbay", path)
}

// readEnvString reads key from the environment, returns def if empty
func readEnvString(key, def string) string {
	if env := os.Getenv(key); env != "" {
		return env
	}
	return def
}

// readEnvBool read key form the env, converting to a boolean value. returns def if empty
func readEnvBool(key string, def bool) bool {
	if env := os.Getenv(key); env != "" {
		return env == "true" || env == "TRUE" || env == "t"
	}
	return def
}

// readEnvString reads a slice of strings from key environment var, returns def if empty
func readEnvStringSlice(key string, def []string) []string {
	if env := os.Getenv(key); env != "" {
		return strings.Split(env, ",")
	}
	return def
}

// requireConfigStrings panics if any of the passed in values aren't set
func requireConfigStrings(values map[string]string) error {
	for key, value := range values {
		if value == "" {
			return fmt.Errorf("%s env variable or config key must be set", key)
		}
	}
	return nil
}

// checks for config.[mode].json file to read configuration from if the file exists
// defaults to config.json, silently fails if no configuration file is present.
func loadConfigFile(mode string, cfg *config) (err error) {
	var data []byte

	fileName := packagePath(fmt.Sprintf("config.%s.json", mode))
	if !fileExists(fileName) {
		fileName = "config.json"
		if !fileExists(fileName) {
			return nil
		}
	}

	logger.Printf("reading config file: %s", fileName)
	data, err = ioutil.ReadFile(fileName)
	if err != nil {
		err = fmt.Errorf("error reading %s: %s", fileName, err)
		return
	}

	// unmarshal ("decode") config data into a config struct
	if err = json.Unmarshal(data, cfg); err != nil {
		err = fmt.Errorf("error parsing %s: %s", fileName, err)
		return
	}

	return
}

// Does this file exist?
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// outputs any notable settings to stdout
func printConfigInfo() {
	// TODO
}
