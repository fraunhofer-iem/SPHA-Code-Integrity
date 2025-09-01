package gh

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
)

const baseURL = "https://api.github.com"

type Result struct {
	Entry string
}

// Create an HTTP request using the parameters, token, method.
func createHttpRequest(reqUrl, requestMethod string, requestBody io.ReadCloser, queryParameters, headerParameters map[string]string) (*http.Request, error) {

	req, err := http.NewRequest(requestMethod, reqUrl, requestBody)
	if err != nil {
		return req, err
	}

	for key, value := range headerParameters {
		req.Header.Set(key, value)
	}

	query := req.URL.Query()

	for key, value := range queryParameters {
		query.Add(key, value)
	}

	req.URL.RawQuery = query.Encode()

	return req, nil
}

// Update an existing HTTP request using the parameters, token and method.
func updateHttpRequest(reqUrl, requestMethod string, requestBody io.ReadCloser, queryParameters, headerParameters map[string]string, req *http.Request) error {

	parsedUrl, err := url.Parse(reqUrl)
	if err != nil {
		slog.Default().Error("Error while parsing the URL - %v ", "error", err)
		return err
	}

	query := parsedUrl.Query()
	req.Method = requestMethod
	req.Body = requestBody

	for key, value := range headerParameters {
		req.Header.Set(key, value)
	}

	for key, value := range queryParameters {
		if query.Has(key) {
			query.Set(key, value)
		} else {
			query.Add(key, value)
		}

	}

	req.URL.RawQuery = query.Encode()

	return nil
}

// Entry point for the Force Push processing, setting the url, query paramenters and creating the client.
func GetForcePushInfo(owner, repo, token, branch string) (int, error) {

	slog.Default().Info("Getting repo info - getForcePushInfo method")
	repoActivityUrl := baseURL + "/repos/" + owner + "/" + repo + "/activity"

	queryParameters := map[string]string{
		"per_page":      "100",
		"activity_type": "force_push",
		"ref":           branch,
	}

	headerParameters := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	}

	client := &http.Client{}

	numberForcePush, err := processForcePushRequest(client, repoActivityUrl, queryParameters, headerParameters)
	if err != nil {
		slog.Default().Error("Getting repo info falied - %v getForcePushInfo method", "error", err)
		return 0, err
	}
	return numberForcePush, nil

}

// Execute the HTTP request and return response
func executeHTTPRequest(client *http.Client, req *http.Request) (*http.Response, error) {

	resp, err := client.Do(req)
	RequestCounter++
	if err != nil {
		return nil, err
	}
	return resp, nil

}

// Handle calls to other methods (createHTTPRequest, executeHTTPRequest, processHttpResponse) is handled and loop is added so that requests are processed till there no rel = "next"
func processForcePushRequest(client *http.Client, repoActivityUrl string, queryParameters, headerParameters map[string]string) (int, error) {

	numberForcePush := 0
	hasNext := true

	// Create an initial HTTP Request and set header and query parameters
	httpReq, err := createHttpRequest(repoActivityUrl, "GET", nil, queryParameters, headerParameters)
	if err != nil {
		slog.Default().Error("Failed to create HTTP request: %v", "error", err)
		return 0, err
	}

	for hasNext {

		// Execute the HTTP Request
		resp, err := executeHTTPRequest(client, httpReq)
		if err != nil {
			slog.Default().Error("Failed to execute HTML request: %v - processForcePushRequest method", "error", err)
			return 0, err
		}

		defer resp.Body.Close()

		// Process the response as result data type
		res, err := processHttpResponse(resp)
		if err != nil {
			slog.Default().Error("Failed to process HTTP response: %v - processForcePushRequest method", "error", err)
			return 0, err
		}

		numberForcePush += len(res)

		// Check if Next Page exists
		hasNext, err = checkIfNextPageExists(resp)
		if err != nil {
			slog.Default().Error("Failed to check for next pages in force pushes: %v - processHttpResponse method", "error", err)
			return 0, err
		}

		// Break from the loop if there is no next reference in the header
		if !hasNext {
			break
		}

		repoActivityUrl, err = getNextURL(resp)
		if err != nil {
			slog.Default().Error("Failed to get Net Url: %v - processHttpResponse method", "error", err)
			return 0, err
		}

		// Update the exisiting http request
		err = updateHttpRequest(repoActivityUrl, "GET", nil, nil, headerParameters, httpReq)
		if err != nil {
			slog.Default().Error("Failed to update the Http Request: %v - processHttpResponse method", "error", err)
			return 0, err
		}

	}

	return numberForcePush, nil

}

// Decode the response and assign it to the Result type struct, we are only interested in the array length of the result since we are only quering the force pushes. After getting the array length check the header for next page.
func processHttpResponse(resp *http.Response) ([]Result, error) {

	var res []Result

	decoder := json.NewDecoder(resp.Body)

	err := decoder.Decode(&res)
	if err != nil {
		slog.Default().Error("Failed to decode JSON response: %v - processHttpResponse method", "error", err)
		return nil, err
	}

	return res, nil
}

// Check if the header of the response contains rel = "next", return a boolean beased on the check
func checkIfNextPageExists(resp *http.Response) (bool, error) {

	linkHeader := resp.Header.Get("Link")
	hasNext := "rel=\"next\""

	hasNextMatch, err := regexp.Match(hasNext, []byte(linkHeader))
	if err != nil {
		slog.Default().Error("Failed to check for Link Header: %v - checkIfNextPageExists method", "error", err)
		return false, err
	}

	return hasNextMatch, nil

}

// Get the URL for the next HTTP request if rel = "next" exists.
func getNextURL(resp *http.Response) (string, error) {

	linkHeader := resp.Header.Get("Link")
	nextUrl := "<([^{}]*)>; rel=\"next\""
	nextPatternMatch := regexp.MustCompile(nextUrl)
	findString := nextPatternMatch.FindString(linkHeader)

	return findString[1 : len(findString)-13], nil

}
