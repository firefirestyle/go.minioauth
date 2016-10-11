package facebook

import (
	"net/http"

	"encoding/json"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

type FaceBook struct {
	AppId                string
	AppSecret            string
	OAuthDialogAddr      string //https://www.facebook.com/dialog/oauth
	OAuthAccessTokenAddr string //"https://graph.facebook.com/oauth/access_token"
}

func NewFaceBook(appId string, appSecret string) *FaceBook {
	ret := new(FaceBook)
	ret.AppId = appId
	ret.AppSecret = appSecret
	ret.OAuthDialogAddr = "https://www.facebook.com/dialog/oauth"
	ret.OAuthAccessTokenAddr = "https://graph.facebook.com/v2.3/oauth/access_token" //https://graph.facebook.com/oauth/access_token"
	return ret
}

func (obj *FaceBook) GetRequestToken(redirectUrl string) string {
	//, , obj.OAuthAccessTokenAddr
	oauth := NewOAuth2Client(obj.AppId, obj.AppSecret) //, obj.OAuthDialogAddr, obj.OAuthAccessTokenAddr)
	return oauth.GetRequestToken(obj.OAuthDialogAddr, redirectUrl)
}

func (obj *FaceBook) CallbackFaceBook(w http.ResponseWriter, r *http.Request, redirectUrl string) (*AccessTokenResponse, error) {
	ctx := appengine.NewContext(r)
	//
	//
	oauth := NewOAuth2Client(obj.AppId, obj.AppSecret)
	//
	code := r.FormValue("code")
	return oauth.RequestAccessToken(ctx, obj.OAuthAccessTokenAddr, redirectUrl, code)
}

type GetMeResponse struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func (obj *FaceBook) GetMe(ctx context.Context, accessToken string) (*GetMeResponse, error) {
	oauth := NewOAuth2Client(obj.AppId, obj.AppSecret)
	response, err := oauth.RequestAPI(ctx, "https://graph.facebook.com/me", accessToken)
	if err != nil {
		return nil, err
	}
	log.Infof(ctx, "> GetMe>"+string(response))
	userInfoBase := new(GetMeResponse)
	json.Unmarshal(response, userInfoBase)
	//

	return userInfoBase, nil
}
