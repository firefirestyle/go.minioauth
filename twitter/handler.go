package twitter

import (
	"net/http"
	"net/url"

	"errors"

	"google.golang.org/appengine"
	//	"google.golang.org/appengine/log"
)

const (
	UrlOptCallbackUrl              = "cb"
	UrlOptErrorNotFoundCallbackUrl = "1001"
	UrlOptErrorFailedToMakeToken   = "1002"
)

type TwitterOAuthConfig struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

type TwitterHandler struct {
	twitterManager *TwitterManager
	onRequest      func(url.Values) map[string]string
	onFoundUser    func(url.Values, *SendAccessTokenResult) map[string]string
	callbackUrl    string
}

type TwitterHundlerOnEvent struct {
	OnRequest   func(url.Values) map[string]string
	OnFoundUser func(url.Values, *SendAccessTokenResult) map[string]string
}

func NewTwitterHandler(callbackUrl string, //
	config TwitterOAuthConfig, //
	onEvent TwitterHundlerOnEvent) *TwitterHandler {
	twitterHandlerObj := new(TwitterHandler)
	twitterHandlerObj.callbackUrl = callbackUrl
	twitterHandlerObj.twitterManager = NewTwitterManager( //
		config.ConsumerKey, config.ConsumerSecret, config.AccessToken, config.AccessTokenSecret)
	twitterHandlerObj.onFoundUser = onEvent.OnFoundUser
	twitterHandlerObj.onRequest = onEvent.OnRequest
	return twitterHandlerObj
}

func (obj *TwitterHandler) MakeUrlNotFoundCallbackError(baseAddr string) (string, error) {
	urlObj, err := url.Parse(baseAddr)
	if err != nil {
		return "", err
	}
	query := urlObj.Query()
	query.Add("error", UrlOptErrorNotFoundCallbackUrl)
	urlObj.RawQuery = query.Encode()
	return urlObj.String(), nil
}

func (obj *TwitterHandler) MakeUrlFailedToMakeToken(baseAddr string) (string, error) {
	urlObj, err := url.Parse(baseAddr)
	if err != nil {
		return "", err
	}
	query := urlObj.Query()
	query.Add("error", UrlOptErrorFailedToMakeToken)
	urlObj.RawQuery = query.Encode()
	return urlObj.String(), nil
}

func (obj *TwitterHandler) TwitterLoginEntry(w http.ResponseWriter, r *http.Request) {
	clCallbackUrl := r.URL.Query().Get(UrlOptCallbackUrl)

	//
	// make redirect URL
	if clCallbackUrl == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//
	clCallbackUrlObj, clCallbackUrlErr := url.Parse(clCallbackUrl)
	if clCallbackUrlErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//
	svCallbackUrlObj, _ := url.Parse(obj.callbackUrl)
	if svCallbackUrlObj.Path == clCallbackUrlObj.Path {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//
	tmpValues := svCallbackUrlObj.Query()
	tmpValues.Add(UrlOptCallbackUrl, clCallbackUrl)
	if obj.onRequest != nil {
		opts := obj.onRequest(r.URL.Query())
		for k, v := range opts {
			tmpValues.Add(k, v)
		}
	}
	svCallbackUrlObj.RawQuery = tmpValues.Encode()
	//
	//
	redirectUrl := ""

	twitterObj := obj.twitterManager.NewTwitter()
	oauthResult, err := twitterObj.SendRequestToken(appengine.NewContext(r), svCallbackUrlObj.String())
	if err != nil {
		failedOAuthUrl, _ := obj.MakeUrlFailedToMakeToken(clCallbackUrl)
		redirectUrl = failedOAuthUrl
	} else {
		redirectUrl = oauthResult.GetOAuthTokenUrl()
	}
	//
	// Do Redirect
	http.Redirect(w, r, redirectUrl, http.StatusFound)
}

func (obj *TwitterHandler) TwitterLoginExit(w http.ResponseWriter, r *http.Request) {
	//
	//
	callbackUrl := r.URL.Query().Get(UrlOptCallbackUrl)
	urlObj, err := url.Parse(callbackUrl)
	if err != nil {
		removeUrlObj, _ := url.Parse(r.RemoteAddr)
		query := removeUrlObj.Query()
		query.Add("error", "error")
		removeUrlObj.RawQuery = query.Encode()
		http.Redirect(w, r, removeUrlObj.String(), http.StatusFound)
		return
	}

	//
	//
	twitterObj := obj.twitterManager.NewTwitter()
	rt, err := twitterObj.OnCallbackSendRequestToken(appengine.NewContext(r), r.URL)
	if err != nil || rt.GetScreenName() == "" || rt.GetUserID() == "" {
		rt = nil
		if err == nil && (rt.GetScreenName() == "" || rt.GetUserID() == "") {
			err = errors.New("empty user")
		}
	}

	if obj.onFoundUser != nil {
		values := urlObj.Query()
		opts := obj.onFoundUser(r.URL.Query(), rt)
		for k, v := range opts {
			values.Add(k, v)
		}
		urlObj.RawQuery = values.Encode()
	}
	//

	if err != nil {
		query := urlObj.Query()
		query.Add("error", "oauth")
		urlObj.RawQuery = query.Encode()
		http.Redirect(w, r, urlObj.String(), http.StatusFound)
	} else {
		http.Redirect(w, r, urlObj.String(), http.StatusFound)
	}
}
