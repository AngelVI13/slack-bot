package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

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

	HcmQdevUrl      string
	HcmQuadUrl      string
	HcmApiToken     string
	HcmHashFilename string
}

// ConfigFromEnv Creates config instance by reading corresponding ENV variables.
// Make sure godotenv.Load is called beforehand
func NewConfigFromEnv(envPath string) *Config {
	// Env variables are used to configure slack client, devices, spaces & users data
	godotenv.Load(envPath)

	taEndpoint := os.Getenv("SL_TA_ENDPOINT")

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

		HcmQdevUrl:      os.Getenv("HCM_QDEV_URL"),
		HcmQuadUrl:      os.Getenv("HCM_QUAD_URL"),
		HcmApiToken:     os.Getenv("HCM_API_TOKEN"),
		HcmHashFilename: os.Getenv("HCM_HASH_FILE"),
	}
}
