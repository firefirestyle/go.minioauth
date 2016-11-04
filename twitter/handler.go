package twitter

import (
	"net/http"
	"net/url"

	"errors"

	"crypto/sha1"
	"encoding/base64"
	"io"

	"strings"

	"github.com/firefirestyle/go.miniprop"
	"google.golang.org/appengine"
	//	"google.golang.org/appengine/log"
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
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
	CallbackUrl       string
	SecretSign        string
}

type TwitterHandler struct {
	twitterManager *TwitterManager
	config         TwitterOAuthConfig
	onEvent        TwitterHundlerOnEvent
}

type TwitterHundlerOnEvent struct {
	OnRequest   func(http.ResponseWriter, *http.Request, *TwitterHandler) (map[string]string, error)
	OnFoundUser func(http.ResponseWriter, *http.Request, *TwitterHandler, *SendAccessTokenResult) map[string]string
}

func NewTwitterHandler( //
	config TwitterOAuthConfig, //
	onEvent TwitterHundlerOnEvent) *TwitterHandler {
	twitterHandlerObj := new(TwitterHandler)
	//	twitterHandlerObj.callbackUrl = callbackUrl
	twitterHandlerObj.twitterManager = NewTwitterManager( //
		config.ConsumerKey, config.ConsumerSecret, config.AccessToken, config.AccessTokenSecret)
	twitterHandlerObj.config = config

	//
	//
	if onEvent.OnRequest == nil {
		onEvent.OnRequest = func(http.ResponseWriter, *http.Request, *TwitterHandler) (map[string]string, error) {
			return map[string]string{}, nil
		}
	}
	if onEvent.OnFoundUser == nil {
		onEvent.OnFoundUser = func(http.ResponseWriter, *http.Request, *TwitterHandler, *SendAccessTokenResult) map[string]string {
			return map[string]string{}
		}
	}
	twitterHandlerObj.onEvent = onEvent
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

func (obj *TwitterHandler) HandleLoginEntry(w http.ResponseWriter, r *http.Request) {
	clCallbackUrl := r.URL.Query().Get(UrlOptCallbackUrl)
	//ctx := appengine.NewContext(r)
	//Debug(ctx, "HandleLoginEntry")
	//
	// make redirect URL
	if clCallbackUrl == "" {
		//Debug(ctx, "HandleLoginEntry callback error")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//
	clCallbackUrlObj, clCallbackUrlErr := url.Parse(clCallbackUrl)
	if clCallbackUrlErr != nil {
		//Debug(ctx, "HandleLoginEntry parse error")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//
	//
	opts, optsErr := obj.onEvent.OnRequest(w, r, obj)
	if optsErr != nil {
		//Debug(ctx, "HandleLoginEntry onRequest err")
		tmpValues := clCallbackUrlObj.Query()
		if opts != nil {
			for k, v := range opts {
				tmpValues.Add(k, v)
			}
		}
		clCallbackUrlObj.RawQuery = tmpValues.Encode()
		http.Redirect(w, r, clCallbackUrlObj.String(), http.StatusFound)
		return
	} else {
		//
		svCallbackUrlObj, _ := url.Parse(obj.config.CallbackUrl)
		if svCallbackUrlObj.Path == clCallbackUrlObj.Path {
			//Debug(ctx, "HandleLoginEntry config parse err")

			w.WriteHeader(http.StatusBadRequest)
			return
		}
		//
		tmpValues := svCallbackUrlObj.Query()
		tmpValues.Add(UrlOptCallbackUrl, clCallbackUrl)

		if opts != nil {
			for k, v := range opts {
				tmpValues.Add(k, v)
			}
		}
		//
		{
			publicSign := miniprop.MakeRandomId()
			tmpValues.Add("ps", publicSign)

			hash := sha1.New()
			io.WriteString(hash, publicSign)
			io.WriteString(hash, opts["kw"])
			io.WriteString(hash, opts["kv"])
			io.WriteString(hash, clCallbackUrlObj.String())
			io.WriteString(hash, obj.config.SecretSign)
			calcHash := base64.StdEncoding.EncodeToString(hash.Sum(nil))
			tmpValues.Add("hash", calcHash)
		}
		//
		svCallbackUrlObj.RawQuery = tmpValues.Encode()
		//
		//
		redirectUrl := ""

		twitterObj := obj.twitterManager.NewTwitter()
		oauthResult, err := twitterObj.SendRequestToken(appengine.NewContext(r), svCallbackUrlObj.String())
		if err != nil {
			//Debug(ctx, "HandleLoginEntry make token errr :"+err.Error())

			failedOAuthUrl, _ := obj.MakeUrlFailedToMakeToken(clCallbackUrl)
			redirectUrl = failedOAuthUrl
		} else {
			//Debug(ctx, "HandleLoginEntry config ok log :"+oauthResult.GetOAuthTokenUrl())

			redirectUrl = oauthResult.GetOAuthTokenUrl()
		}
		//
		// Do Redirect
		http.Redirect(w, r, redirectUrl, http.StatusFound)
	}
}

func (obj *TwitterHandler) HandleLoginExit(w http.ResponseWriter, r *http.Request) {
	//
	//
	// response easy check
	{
		values := r.URL.Query()
		hashV := values.Get("hash")
		publicSign := values.Get("ps")
		kw := values.Get("kw")
		kv := values.Get("kv")
		clCallback := values.Get("cb")
		{
			hash := sha1.New()
			io.WriteString(hash, publicSign)
			io.WriteString(hash, kw)
			io.WriteString(hash, kv)
			io.WriteString(hash, clCallback)
			io.WriteString(hash, obj.config.SecretSign)
			calcHash := base64.StdEncoding.EncodeToString(hash.Sum(nil))

			if strings.Compare(calcHash, hashV) != 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
	}
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

	if obj.onEvent.OnFoundUser != nil {
		values := urlObj.Query()
		opts := obj.onEvent.OnFoundUser(w, r, obj, rt)
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

func Debug(ctx context.Context, message string) {
	log.Infof(ctx, message)
}
