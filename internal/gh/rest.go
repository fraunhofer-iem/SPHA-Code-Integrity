package gh

import (
	"net/http"
    "regexp"
    "log/slog"
    "encoding/json"
)

const baseURL = "https://api.github.com"
var NumberForcePush = 0

type Result struct {
    Entry string
}

func createHttpRequest(client *http.Client, url string, token string, queryParameters map[string] string)(*http.Request, error) {
/*
    Create an HTTP request using the parameters, token and http client.
*/

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
        
    req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
    
    query := req.URL.Query()

    for key, value := range queryParameters{
        query.Add(key, value)
    }
    req.URL.RawQuery = query.Encode()

    return req, nil
}

func GetForcePushInfo(owner string, repo string, token string, branch string) (int, error){
/*
    Entry point for the Force Push processing, setting the url, query paramenters and creating the client.
*/
    slog.Default().Info("Getting repo info - getForcePushInfo method")
    repoActivityUrl := baseURL + "/repos/" + owner + "/" + repo + "/activity"
    queryParameters := map[string]string{
        "per_page" : "100",
        "activity_type" : "force_push",
        "ref" : branch,
    }

    client := &http.Client{}

    err := processForcePushRequest(client, repoActivityUrl, token, queryParameters) 
    if err != nil {
        slog.Default().Info("Getting repo info falied - %v getForcePushInfo method", err)
        return 0, err
    }    
    return NumberForcePush, nil
}

func executeHTTPRequest(client *http.Client, req *http.Request) (*http.Response, error){
/*
    Execute the HTTP request and return response
*/

    resp, err := client.Do(req)
	RequestCounter++
	if err != nil {
        return nil, err
	}
    return resp, nil
}

func processForcePushRequest(client *http.Client, repoActivityUrl string, token string, queryParameters map[string] string) error {
/*
    Method where calls to other methods (createHTTPRequest, executeHTTPRequest, processHttpResponse) is handled and loop is added so that requests are processed till there no rel = "next"
*/

    slog.Default().Info("Getting Force Push Info - processForcePushRequestAndResponse method")
    
    for true{
        httpReq, err := createHttpRequest(client, repoActivityUrl, token, queryParameters)
        if err != nil {
	        slog.Default().Error("Failed to create HTML request: %v", err)
	        return  err
        }
        resp, err := executeHTTPRequest(client, httpReq)
        if err != nil {
		    slog.Default().Error("Failed to execute HTML request: %v - processForcePushRequest method", err)
		    return err
	    }
        defer resp.Body.Close()
        repoActivityUrl, err = processHttpResponse(resp)
        if err != nil {
	        slog.Default().Error("Failed to process HTTP response: %v - processForcePushRequest method", err)
	        return err
	    }
        if (len(repoActivityUrl) == 0){
            break
        }
    }
    return nil
}

func processHttpResponse(resp *http.Response)(string, error){
/*
    Decode the response and assign it to the Result type struct, we are only interested in the array length of the result since we are only quering the force pushes. After getting the array length check the header for next page.
*/
    hasNext := false
    var res []Result

    decoder := json.NewDecoder(resp.Body)
    err := decoder.Decode(&res)
    if err != nil {
		slog.Default().Error("Failed to decode JSON response: %v - processHttpResponse method", err)
        return "", err
	}
    NumberForcePush += len(res)
    hasNext, err = checkIfNextPageExists(resp)
    if err != nil {
	    slog.Default().Error("Failed to check for next pages in force pushes: %v - processHttpResponse method", err)
	    return "", err
	}

    if (hasNext){
        nextUrl, err := getNextURL(resp)
        if err != nil {
	        slog.Default().Error("Failed to get Net Url: %v - processHttpResponse method", err)
	        return "", err
	    }
        return nextUrl, nil
    }
    return "", nil
    
}

func checkIfNextPageExists(resp *http.Response) (bool, error){  
/*
    Check if the header of the response contains rel = "next", return a boolean beased on the check
*/
    linkHeader := resp.Header.Get("Link")
    hasNext := "rel=\"next\""
    hasNextMatch, err := regexp.Match(hasNext, []byte(linkHeader))
    if err != nil {
	    slog.Default().Error("Failed to check for Link Header: %v - checkIfNextPageExists method", err)
	    return false, err
	}
    return hasNextMatch, nil
}

func getNextURL(resp *http.Response)(string, error){ 
/*
    Get the URL for the next HTTP request if rel = "next" exists.
*/
    linkHeader := resp.Header.Get("Link")
    nextUrl := "<([^{}]*)>; rel=\"next\""       
    nextPatternMatch := regexp.MustCompile(nextUrl)
    findString := nextPatternMatch.FindString(linkHeader)
    return findString[1 : len(findString)-13], nil
}



