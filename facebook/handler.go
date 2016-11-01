package facebook

import (
	"net/http"

	"io"
	"net/url"

	"crypto/sha1"

	"encoding/base64"

	"strings"

	"github.com/firefirestyle/go.miniprop"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

const (
	UrlOptCallbackUrl              = "cb"
	UrlOptErrorNotFoundCallbackUrl = "1001"
	UrlOptErrorFailedToMakeToken   = "1002"
)

type FacebookOAuthConfig struct {
	ConfigFacebookAppSecret string
	ConfigFacebookAppId     string
	CallbackUrl             string
	SecretSign              string
}

type FacebookHandler struct {
	facebookObj *Facebook
	onEvent     FacebookHundlerOnEvent
	config      FacebookOAuthConfig
}

type FacebookHundlerOnEvent struct {
	OnRequest   func(http.ResponseWriter, *http.Request, *FacebookHandler) (map[string]string, error)
	OnFoundUser func(http.ResponseWriter, *http.Request, *FacebookHandler, *GetMeResponse, *AccessTokenResponse) map[string]string
}

func NewFacebookHandler(config FacebookOAuthConfig, onEvent FacebookHundlerOnEvent) *FacebookHandler {
	if onEvent.OnRequest == nil {
		onEvent.OnRequest = func(http.ResponseWriter, *http.Request, *FacebookHandler) (map[string]string, error) {
			return map[string]string{}, nil
		}
	}
	if onEvent.OnFoundUser == nil {
		onEvent.OnFoundUser = func(http.ResponseWriter, *http.Request, *FacebookHandler, *GetMeResponse, *AccessTokenResponse) map[string]string {
			return map[string]string{}
		}
	}
	return &FacebookHandler{
		facebookObj: NewFacebook(config.ConfigFacebookAppId, config.ConfigFacebookAppSecret),
		onEvent:     onEvent,
		config:      config,
	}
}

func (obj *FacebookHandler) HandleLoginEntry(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	var outerOpts map[string]string = nil
	var outerErr error = nil
	if obj.onEvent.OnRequest != nil {
		outerOpts, outerErr = obj.onEvent.OnRequest(w, r, obj)
	}
	//
	clCallbackUrlObj, clCallbackUrlErr := url.Parse(r.URL.Query().Get(UrlOptCallbackUrl))
	if clCallbackUrlErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//
	if outerErr != nil {
		tmpValues := clCallbackUrlObj.Query()
		if outerOpts != nil {
			for k, v := range outerOpts {
				tmpValues.Add(k, v)
			}
		}
		clCallbackUrlObj.RawQuery = tmpValues.Encode()
		http.Redirect(w, r, clCallbackUrlObj.String(), http.StatusFound)
		return
	} else {

		//
		//
		svCallbackUrlObj, _ := url.Parse(obj.config.CallbackUrl)
		if svCallbackUrlObj.Path == clCallbackUrlObj.Path {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		tmpValues := svCallbackUrlObj.Query()
		tmpValues.Add(UrlOptCallbackUrl, clCallbackUrlObj.String())

		if outerOpts != nil {
			for k, v := range outerOpts {
				tmpValues.Add(k, v)
			}
		}
		//
		{
			publicSign := miniprop.MakeRandomId()
			tmpValues.Add("ps", publicSign)

			hash := sha1.New()
			io.WriteString(hash, publicSign)
			io.WriteString(hash, outerOpts["kw"])
			io.WriteString(hash, outerOpts["kv"])
			io.WriteString(hash, clCallbackUrlObj.String())
			io.WriteString(hash, obj.config.SecretSign)
			calcHash := base64.StdEncoding.EncodeToString(hash.Sum(nil))
			tmpValues.Add("hash", calcHash)
		}
		//
		svCallbackUrlObj.RawQuery = tmpValues.Encode()
		oauthUrl := obj.facebookObj.GetRequestToken(svCallbackUrlObj.String())
		http.Redirect(w, r, oauthUrl, http.StatusFound)
		return
	}

}

func (obj *FacebookHandler) HandleLoginExit(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	ctx := appengine.NewContext(r)
	facebookObj := obj.facebookObj
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

	//----
	// response oauth check
	//
	svCallbackUrlObj, svCallbackUrlErr := url.Parse(obj.config.CallbackUrl)
	if svCallbackUrlErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	callbackAddr := svCallbackUrlObj.Scheme + "://" + svCallbackUrlObj.Host + r.RequestURI

	tokk, errToken := facebookObj.CallbackFaceBook(w, r, callbackAddr)
	if errToken != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	meObj, errMe := facebookObj.GetMe(ctx, tokk.AccessToken)
	if errMe != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//---
	// response owner check
	//
	clCallbackObj, _ := url.Parse(r.URL.Query().Get("cb"))
	tmpValues := clCallbackObj.Query()
	kv := obj.onEvent.OnFoundUser(w, r, obj, meObj, tokk)
	for k, v := range kv {
		tmpValues.Add(k, v)
	}
	clCallbackObj.RawQuery = tmpValues.Encode()
	http.Redirect(w, r, clCallbackObj.String(), http.StatusFound)
}

func Debug(ctx context.Context, message string) {
	log.Infof(ctx, message)
}
