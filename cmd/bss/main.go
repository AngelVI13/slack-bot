package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"

	"github.com/AngelVI13/slack-bot/pkg/bss"
	"github.com/joho/godotenv"
)

type BssTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func searchOperations(bssConfig BssConfig, tokens BssTokens) {
	fullURL := bssConfig.url + "/staff/operations/:search"
	// fullURL := bssConfig.url + "/staff/contracts/:search"

	data := map[string]any{
		"Filtering": map[string]any{
			"Filters": []map[string]any{
				{
					"Field":    "statusCfgNr",
					"Value":    2,
					"operator": "equal",
				},
				{
					"Field":    "ValidFrom",
					"Value":    "2025-09-15",
					"operator": "greaterOrEqual",
				},
				{
					"Field":    "ValidTo",
					"Value":    "2025-10-01",
					"operator": "lessOrEqual",
				},
			},
		},
		"sorting": []map[string]string{
			{
				"field":     "recordCreationDate",
				"direction": "desc",
			},
		},
	}

	b, err := json.Marshal(&data)
	// fmt.Println(fullURL, string(b))
	if err != nil {
		log.Fatalf("Failed to marshal login request body: %v\n%v", data, err)
	}

	respStr := makeRequest(fullURL, tokens.AccessToken, bytes.NewBuffer(b))
	// fmt.Println(respStr)
	_ = respStr
}

func login(bssConfig BssConfig) BssTokens {
	fullURL := bssConfig.url + "/auth"
	data := map[string]any{
		"username":      bssConfig.username,
		"password":      bssConfig.password,
		"environmentId": bssConfig.envID,
		"companyId":     bssConfig.companyID,
	}

	b, err := json.Marshal(&data)
	fmt.Println(fullURL, string(b))
	if err != nil {
		log.Fatalf("Failed to marshal login request body: %v\n%v", data, err)
	}

	respStr := makeRequest(fullURL, "", bytes.NewBuffer(b))
	// fmt.Println(respStr)
	var tokens BssTokens
	err = json.Unmarshal([]byte(respStr), &tokens)
	if err != nil {
		log.Fatalf("failed to unmarshal token response: %v", err)
	}

	// fmt.Println(tokens)
	return tokens
}

func makeRequest(fullURL, token string, body io.Reader) string {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, fullURL, body)
	if err != nil {
		log.Fatal(err)
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	// NOTE: if this is missing the the reply is in XML format
	// Might be more useful to use the XML format because it contains escape codes
	// For lithuanian alphabet special characters whereas json returns the literal characters
	// Might be easiest if i replace the xml espace codes with `.` and perform a regex search to match
	// a user in the parking bot users.json
	// req.Header.Set("Accept", "application/json")

	reqDump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n\nREQUEST:\n%s\n\n", string(reqDump))

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	respDump, err := httputil.DumpResponse(res, true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n\nRESPONSE:\n%s\n\n", string(respDump))

	// fmt.Println(res.StatusCode)
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	return string(b)
}

type BssConfig struct {
	url       string
	username  string
	password  string
	envID     int
	companyID int
}

func NewBssConfig(dotfile string) BssConfig {
	godotenv.Load(dotfile)
	envIDstr := os.Getenv("BSS_ENVIRONMENT_ID")
	envID, err := strconv.Atoi(envIDstr)
	if err != nil {
		log.Fatalf("Failed to convert BSS_ENVIRONMENT_ID to int: %q; %v", envIDstr, err)
	}

	companyIDstr := os.Getenv("BSS_COMPANY_ID")
	companyID, err := strconv.Atoi(companyIDstr)
	if err != nil {
		log.Fatalf("Failed to convert BSS_COMPANY_ID to int: %q; %v", companyIDstr, err)
	}
	return BssConfig{
		url:      os.Getenv("BSS_URL"),
		username: os.Getenv("BSS_USERNAME"),
		// TODO: encrypt and decrypt the password so
		// it's not stored in plaintext inside bss.env file
		password:  os.Getenv("BSS_PASSWORD"),
		envID:     envID,
		companyID: companyID,
	}
}

func parseBssResponse(filename string) {
	b, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	var resp bss.Response
	err = json.Unmarshal(b, &resp)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp)
}

//	curl -X POST "https://erp2.bss.biz/api/auth/" \
//	 -H 'accept: application/json'\
//	 -H 'content-type: application/json' \
//	 -d '{"username":"A","password":"A","environmentId":1,"companyId":1}'
func main() {
	bssConfig := NewBssConfig("bss.env")
	tokens := login(bssConfig)
	searchOperations(bssConfig, tokens)
	// parseBssResponse("bss_operations.json")
}
