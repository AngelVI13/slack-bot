package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func listOfAllEmployees(url, token string) string {
	fullUrl := url + "/ext/api/v1/employees"
	return makeRequest(fullUrl, token)
}

func vacationsOfAllEmployees(url, token string) string {
	fullUrl := url + "/ext/api/v1/employees/periods"
	return makeRequest(fullUrl, token)
}

func businessTripsOfAllEmployees(url, token string) string {
	fullUrl := url + "/ext/api/v1/employees/businesstrips"
	return makeRequest(fullUrl, token)
}

func makeRequest(fullUrl, token string) string {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, fullUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("x-api-key", token)
	// NOTE: if this is missing the the reply is in XML format
	// Might be more useful to use the XML format because it contains escape codes
	// For lithuanian alphabet special characters whereas json returns the literal characters
	// Might be easiest if i replace the xml espace codes with `.` and perform a regex search to match
	// a user in the parking bot users.json
	// req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	return string(b)
}

func main() {
	godotenv.Load(".env")
	token := os.Getenv("HCM_API_TOKEN")
	hcmUrl := os.Getenv("HCM_URL")
	fmt.Println(listOfAllEmployees(hcmUrl, token))
	fmt.Println(vacationsOfAllEmployees(hcmUrl, token))
}
