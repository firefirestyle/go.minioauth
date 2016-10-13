package twitter

import (
	"net/http"
	"net/url"

	"errors"

	"google.golang.org/appengine"
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

func NewTwitterHandler(callbackUrl string, //
	config TwitterOAuthConfig, //
	onRequest func(url.Values) map[string]string,
	onFoundUser func(url.Values, *SendAccessTokenResult) map[string]string) *TwitterHandler {
	twitterHandlerObj := new(TwitterHandler)
	twitterHandlerObj.callbackUrl = callbackUrl
	twitterHandlerObj.twitterManager = NewTwitterManager( //
		config.ConsumerKey, config.ConsumerSecret, config.AccessToken, config.AccessTokenSecret)
	twitterHandlerObj.onFoundUser = onFoundUser
	twitterHandlerObj.onRequest = onRequest
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
	callbackUrl := r.URL.Query().Get(UrlOptCallbackUrl)
	//
	// make redirect URL
	redirectUrl := ""
	if callbackUrl == "" {
		redirectUrl, _ = obj.MakeUrlNotFoundCallbackError(r.RemoteAddr)
	} else {
		//
		callbackUrlObj, _ := url.Parse(obj.callbackUrl)
		tmpValues := callbackUrlObj.Query()
		tmpValues.Add(UrlOptCallbackUrl, callbackUrl)
		if obj.onRequest != nil {
			opts := obj.onRequest(r.URL.Query())
			for k, v := range opts {
				tmpValues.Add(k, v)
			}
		}
		callbackUrlObj.RawQuery = tmpValues.Encode()

		//
		twitterObj := obj.twitterManager.NewTwitter()
		oauthResult, err := twitterObj.SendRequestToken(appengine.NewContext(r), callbackUrlObj.String())
		if err != nil {
			failedOAuthUrl, err := obj.MakeUrlFailedToMakeToken(callbackUrl)
			if err != nil {
				redirectUrl, _ = obj.MakeUrlNotFoundCallbackError(r.RemoteAddr)
			} else {
				redirectUrl = failedOAuthUrl
			}
		} else {
			redirectUrl = oauthResult.GetOAuthTokenUrl()
		}
	}
	//
	// Do Redirect
	http.Redirect(w, r, redirectUrl, http.StatusFound)
}

func (obj *TwitterHandler) TwitterLoginExit(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	ctx := appengine.NewContext(r)
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
	rt, err := twitterObj.OnCallbackSendRequestToken(ctx, r.URL)
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
		return
	} else {
		http.Redirect(w, r, urlObj.String(), http.StatusFound)
	}
}
