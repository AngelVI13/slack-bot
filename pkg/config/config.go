package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type BssCompanyConfig struct {
	Username      string
	Password      string
	EnvironmentId int
	CompanyId     int
}

func NewBssCompanyConfig(
	username, password, envIdStr, companyIdStr string,
) BssCompanyConfig {
	envId, err := strconv.Atoi(envIdStr)
	if err != nil {
		log.Fatalf(
			"Failed to convert BSS ENVIRONMENT_ID to int: %q; %v",
			envIdStr,
			err,
		)
	}

	companyId, err := strconv.Atoi(companyIdStr)
	if err != nil {
		log.Fatalf(
			"Failed to convert BSS COMPANY_ID to int: %q; %v",
			companyIdStr,
			err,
		)
	}
	return BssCompanyConfig{
		Username:      username,
		Password:      password,
		EnvironmentId: envId,
		CompanyId:     companyId,
	}
}

type BssConfig struct {
	Url  string
	Qdev BssCompanyConfig
	Quad BssCompanyConfig
}

func NewBssConfig(url string, qdev, quad BssCompanyConfig) BssConfig {
	return BssConfig{
		Url:  url,
		Qdev: qdev,
		Quad: quad,
	}
}

type Config struct {
	SlackAuthToken   string
	SlackTaChannelId string
	SlackAppToken    string

	DevicesFilename    string
	UsersFilename      string
	ParkingFilename    string
	WorkspacesFilename string

	Debug           bool
	TaEndpoint      string
	WorkersEndpoint string
	ProxyEndpoint   string

	ReportPersonId string

	HcmQdevUrl            string
	HcmQuadUrl            string
	HcmApiToken           string
	VacationsHashFilename string

	Bss BssConfig
}

// NewConfigFromEnv Creates config instance by reading corresponding ENV variables.
// Make sure godotenv.Load is called beforehand
func NewConfigFromEnv(envPath string) *Config {
	// Env variables are used to configure slack client, devices, spaces & users data
	godotenv.Load(envPath)

	taEndpoint := os.Getenv("SL_TA_ENDPOINT")

	bssQuad := NewBssCompanyConfig(
		os.Getenv("BSS_QUAD_USERNAME"),
		os.Getenv("BSS_QUAD_PASSWORD"),
		os.Getenv("BSS_QUAD_ENVIRONMENT_ID"),
		os.Getenv("BSS_QUAD_COMPANY_ID"),
	)

	bssQdev := NewBssCompanyConfig(
		os.Getenv("BSS_QDEV_USERNAME"),
		os.Getenv("BSS_QDEV_PASSWORD"),
		os.Getenv("BSS_QDEV_ENVIRONMENT_ID"),
		os.Getenv("BSS_QDEV_COMPANY_ID"),
	)

	bssConfig := NewBssConfig(
		os.Getenv("BSS_URL"),
		bssQdev,
		bssQuad,
	)

	return &Config{
		SlackAuthToken:   os.Getenv("SLACK_AUTH_TOKEN"),
		SlackTaChannelId: os.Getenv("SLACK_TA_CHANNEL_ID"),
		SlackAppToken:    os.Getenv("SLACK_APP_TOKEN"),

		DevicesFilename: os.Getenv("SL_DEVICES_FILE"),
		UsersFilename:   os.Getenv("SL_USERS_FILE"),

		ParkingFilename:    os.Getenv("SL_PARKING_FILE"),
		WorkspacesFilename: os.Getenv("SL_WORKSPACES_FILE"),

		Debug:           os.Getenv("SL_DEBUG") == "1",
		TaEndpoint:      taEndpoint,
		WorkersEndpoint: fmt.Sprintf("%s/workers", taEndpoint),
		ProxyEndpoint:   fmt.Sprintf("%s/proxy", taEndpoint),

		ReportPersonId: os.Getenv("REPORT_PERSON_ID"),

		HcmQdevUrl:            os.Getenv("HCM_QDEV_URL"),
		HcmQuadUrl:            os.Getenv("HCM_QUAD_URL"),
		HcmApiToken:           os.Getenv("HCM_API_TOKEN"),
		VacationsHashFilename: os.Getenv("HCM_HASH_FILE"),

		Bss: bssConfig,
	}
}
