package facebook

import (
	"net/http"

	"net/url"

	"github.com/firefirestyle/go.minioauth/sns"
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
	AllowInvalidSSL         bool
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
		facebookObj: NewFacebook(config.ConfigFacebookAppId, config.ConfigFacebookAppSecret, config.AllowInvalidSSL),
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
		HandleError(w, r, nil, 3030, clCallbackUrlErr.Error())
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
	}

	//
	//
	svCallbackUrlObj, svCallbackUrlErr := url.Parse(obj.config.CallbackUrl)
	if svCallbackUrlErr != nil {
		HandleError(w, r, nil, 3040, svCallbackUrlErr.Error())
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
	sns.WithHashAndValue(tmpValues, obj.config.SecretSign, clCallbackUrlObj.String(), outerOpts)
	//
	svCallbackUrlObj.RawQuery = tmpValues.Encode()
	oauthUrl := obj.facebookObj.GetRequestToken(svCallbackUrlObj.String())
	http.Redirect(w, r, oauthUrl, http.StatusFound)
	return

}

func (obj *FacebookHandler) HandleLoginExit(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	ctx := appengine.NewContext(r)
	facebookObj := obj.facebookObj
	//
	// response easy check
	clCallbackObj, clCallbackErr := url.Parse(r.URL.Query().Get("cb"))
	if clCallbackErr != nil {
		HandleError(w, r, nil, 3000, "Failed to make clcallback url")
		return
	}
	checkErr := sns.CheckHashAndValue(w, r, obj.config.SecretSign)
	if checkErr != nil {
		HandleError(w, r, nil, 3010, "Failed in hash check")
		return
	}

	//----
	// response oauth check
	//
	svCallbackUrlObj, svCallbackUrlErr := url.Parse(obj.config.CallbackUrl)
	if svCallbackUrlErr != nil {
		HandleError(w, r, nil, 3020, "Failed to make svcallback url")
		return
	}
	callbackAddr := svCallbackUrlObj.Scheme + "://" + svCallbackUrlObj.Host + r.RequestURI

	tokk, errToken := facebookObj.CallbackFaceBook(w, r, callbackAddr)
	if errToken != nil {
		HandleError(w, r, nil, 3030, errToken.Error())
		return
	}
	meObj, errMe := facebookObj.GetMe(ctx, tokk.AccessToken)
	if errMe != nil {
		HandleError(w, r, nil, 3040, errMe.Error())
		return
	}

	//---
	// response owner check
	//
	//	clCallbackObj, _ := url.Parse(r.URL.Query().Get("cb"))
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

func HandleError(w http.ResponseWriter, r *http.Request, outputProp *miniprop.MiniProp, errorCode int, errorMessage string) {
	//
	//
	if outputProp == nil {
		outputProp = miniprop.NewMiniProp()
	}
	if errorCode != 0 {
		outputProp.SetInt("errorCode", errorCode)
	}
	if errorMessage != "" {
		outputProp.SetString("errorMessage", errorMessage)
	}
	w.WriteHeader(http.StatusBadRequest)
	w.Write(outputProp.ToJson())
}
