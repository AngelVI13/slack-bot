package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

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

	HcmQdevUrl            string
	HcmQuadUrl            string
	HcmApiToken           string
	VacationsHashFilename string

	BssUrl               string
	BssQuadUsername      string
	BssQuadPassword      string
	BssQuadEnvironmentId int
	BssQuadCompanyId     int
}

// NewConfigFromEnv Creates config instance by reading corresponding ENV variables.
// Make sure godotenv.Load is called beforehand
func NewConfigFromEnv(envPath string) *Config {
	// Env variables are used to configure slack client, devices, spaces & users data
	godotenv.Load(envPath)

	taEndpoint := os.Getenv("SL_TA_ENDPOINT")

	bssQuadEnvIDstr := os.Getenv("BSS_QUAD_ENVIRONMENT_ID")
	bssQuadEnvID, err := strconv.Atoi(bssQuadEnvIDstr)
	if err != nil {
		log.Fatalf(
			"Failed to convert BSS_QUAD_ENVIRONMENT_ID to int: %q; %v",
			bssQuadEnvIDstr,
			err,
		)
	}

	bssQuadCompanyIDstr := os.Getenv("BSS_QUAD_COMPANY_ID")
	bssQuadCompanyID, err := strconv.Atoi(bssQuadCompanyIDstr)
	if err != nil {
		log.Fatalf(
			"Failed to convert BSS_QUAD_COMPANY_ID to int: %q; %v",
			bssQuadCompanyIDstr,
			err,
		)
	}

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

		BssUrl:               os.Getenv("BSS_URL"),
		BssQuadUsername:      os.Getenv("BSS_QUAD_USERNAME"),
		BssQuadPassword:      os.Getenv("BSS_QUAD_PASSWORD"),
		BssQuadEnvironmentId: bssQuadEnvID,
		BssQuadCompanyId:     bssQuadCompanyID,
	}
}
