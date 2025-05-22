package gh

import (
	"net/http"
    //"fmt"
    "regexp"
    "log/slog"
    "encoding/json"
)

const baseURL = "https://api.github.com"
//var RequestCounter = 0
var NumberForcePush = 0

type Result struct {
    Entry string
}

func createHttpRequest(client *http.Client, url string, token string, queryParameters map[string] string)(*http.Request, error) {
    //fmt.Print("Start of Create Http Request")
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        //fmt.Errorf("Failed to create a new request: %v", err)
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
    slog.Default().Info("Getting repo info - getForcePushInfo method")
    //fmt.Print("Start of Get Force Push")
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
    //fmt.Print("Start of Execute Http Request")
    resp, err := client.Do(req)
	RequestCounter++
	if err != nil {
		//fmt.Errorf("HTTP request failed: %v", err)
        return nil, err
	}
    return resp, nil
}

func processForcePushRequest(client *http.Client, repoActivityUrl string, token string, queryParameters map[string] string) error {

    slog.Default().Info("Getting Force Push Info - processForcePushRequestAndResponse method")
    //fmt.Print("Start of Process Force Push")
    
    for true{
        //fmt.Print("\n\n\nRepoActivityURL - %v\n\n\n", repoActivityUrl)
        httpReq, err := createHttpRequest(client, repoActivityUrl, token, queryParameters)
        if err != nil {
	        slog.Default().Error("Failed to create HTML request: %v", err)
	        return  err
        }
        //fmt.Print("\n Http Request - %v",httpReq)
        resp, err := executeHTTPRequest(client, httpReq)
        if err != nil {
		    slog.Default().Error("Failed to execute HTML request: %v - processForcePushRequest method", err)
		    return err
	    }
        defer resp.Body.Close()
        //fmt.Print("\n Http Response - %v",resp.Body)
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
    //fmt.Print("Start of Process Http Response")
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
    //fmt.Print("\nStart of Check Next Page Exisits")
    linkHeader := resp.Header.Get("Link")
    hasNext := "rel=\"next\""
    hasNextMatch, err := regexp.Match(hasNext, []byte(linkHeader))
    if err != nil {
	    slog.Default().Error("Failed to check for Link Header: %v - checkIfNextPageExists method", err)
	    return false, err
	}
    //fmt.Println(string(linkHeader))    
    //fmt.Print("\nHas Next - ",hasNextMatch)
    return hasNextMatch, nil
}

func getNextURL(resp *http.Response)(string, error){
    //fmt.Print("Start of Get Next Url")
    linkHeader := resp.Header.Get("Link")
    nextUrl := "<([^{}]*)>; rel=\"next\""
    nextPatternMatch := regexp.MustCompile(nextUrl)
    findString := nextPatternMatch.FindString(linkHeader)
    //fmt.Println(findString[1 : len(findString)-13])
    return findString[1 : len(findString)-13], nil
}
/*
func main(){
    
    repo := "Test_011"
    branch :="sub-feature"
    
    
    noForcePush, err := getForcePushInfo(owner, repo, token, branch)
    if err != nil {
		slog.Default().Error("Force Push Info Failed - %v", err)
		
	}
    fmt.Print(noForcePush)

    /*
    url := "https://api.github.com/repos/fraunhofer-iem/SPHA-Code-Integrity/activity?page=1?activity_type=branch_creation"
    req, err := http.NewRequest("GET",url, nil)
    if err != nil {
        fmt.Print(err.Error())
    }
    
    res, err := http.DefaultClient.Do(req)
    if err != nil {
        fmt.Print(err.Error())
    }
    defer res.Body.Close()
    body, readErr := ioutil.ReadAll(res.Body)
    if readErr != nil {
        fmt.Print(readErr.Error())
    }
    //header, err := ioutil.ReadAll(res.Header)
    if err != nil {
        fmt.Print(err.Error())
    }
    fmt.Println(string(body))
    linkHeader := res.Header.Get("Link")
    fmt.Println(string(linkHeader))
    hasNext := "rel=\"next\""
    nextUrl := "<([^{}]*)>"
    fmt.Println(nextUrl)
    hasNextMatch, err := regexp.Match(hasNext, []byte(linkHeader))
    nextPatternMatch := regexp.MustCompile(nextUrl)
    //find := obj.Find([]byte(linkHeader))
    findString := nextPatternMatch.FindString(linkHeader)
    fmt.Println(findString[1 : len(findString)-1])
    fmt.Println(hasNextMatch)

    url = findString[1 : len(findString)-1]
    req, err = http.NewRequest("GET",url, nil)
    if err != nil {
        fmt.Print(err.Error())
    }
    
    res, err = http.DefaultClient.Do(req)
    if err != nil {
        fmt.Print(err.Error())
    }
    defer res.Body.Close()
    body, readErr = ioutil.ReadAll(res.Body)
    if readErr != nil {
        fmt.Print(readErr.Error())
    }
    //header, err := ioutil.ReadAll(res.Header)
    if err != nil {
        fmt.Print(err.Error())
    }
    fmt.Println(string(body))

}
*/



