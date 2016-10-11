package facebook

import (
	"fmt"
	"net/http"

	"bytes"

	//	"strings"

	"errors"
	"net/url"

	"encoding/json"

	//	"io/ioutil"

	"golang.org/x/net/context"
	//	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

// https://developers.facebook.com/docs/facebook-login/manually-build-a-login-flow?locale=en_US
type OAuth2Client struct {
	AppId     string
	AppSecret string
	//OAuthDialogAddr      string
	//OAuthAccessTokenAddr string
}

//, oauthDialogAddr string, oauthAccessTokenAddr string
func NewOAuth2Client(appId string, appSecret string) *OAuth2Client {
	ret := new(OAuth2Client)
	ret.AppId = appId
	ret.AppSecret = appSecret
	//ret.OAuthDialogAddr = oauthDialogAddr
	//ret.OAuthAccessTokenAddr = oauthAccessTokenAddr
	return ret
}

func (obj *OAuth2Client) GetRequestToken(oauthDialogAddr string, redirectUri string) string {
	targetUri := fmt.Sprintf( //
		"%s?client_id=%s&redirect_uri=%s&response_type=code", //
		oauthDialogAddr, url.QueryEscape(obj.AppId), url.QueryEscape(redirectUri))
	return targetUri
}

type AccessTokenResponse struct {
	AccessToken string
	TokenType   string
	ExpiresIn   float64
}

func (obj *OAuth2Client) RequestAccessToken(ctx context.Context, oauthAccessTokenAddr string, redirectUri string, code string) (*AccessTokenResponse, error) {

	//
	//
	targetUri := //
		fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&client_secret=%s&code=%s", //
			oauthAccessTokenAddr, url.QueryEscape(obj.AppId), url.QueryEscape(redirectUri), url.QueryEscape(obj.AppSecret), url.QueryEscape(code))
	//
	//
	request, _ := http.NewRequest(http.MethodPost, targetUri, bytes.NewBufferString(""))
	request.Method = "GET"
	//
	client := urlfetch.Client(ctx)
	res, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	//
	var requestPropery map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&requestPropery)
	if err != nil {
		return nil, err
	}
	if requestPropery["access_token"] == nil {
		t, _ := json.Marshal(requestPropery)
		return nil, errors.New("error response" + string(t))
	}

	ret := new(AccessTokenResponse)
	ret.AccessToken = requestPropery["access_token"].(string)
	ret.ExpiresIn = requestPropery["expires_in"].(float64)
	ret.TokenType = requestPropery["token_type"].(string)
	// {“access_token”: <access-token>, “token_type”:<type>, “expires_in”:<seconds-til-expiration>}
	return ret, nil
}

func (obj *OAuth2Client) RequestAPI(ctx context.Context, targetUri string, accessToken string) ([]byte, error) {
	params := make(map[string]string, 1)
	params["access_token"] = accessToken
	//
	targetUriWithQuery := //
		fmt.Sprintf("%s?access_token=%s", //
			targetUri, url.QueryEscape(accessToken))

	request, _ := http.NewRequest(http.MethodGet, targetUriWithQuery, bytes.NewBufferString(""))
	//
	client := urlfetch.Client(ctx)
	response, _ := client.Do(request)
	result := make([]byte, 1024)
	i, err := response.Body.Read(result)
	if err != nil {
		return make([]byte, 0), err
	}
	result = result[0:i]
	return result, nil
}

/*
https://graph.facebook.com/v2.3/oauth/access_token
?client_id=1242352189120394
&redirect_uri=http://localhost:8080/api/v1/me_mana/facebook/oauth?cb=http%!A(MISSING)%!F(MISSING)%!F(MISSING)127.0.0.1%!A(MISSING)8085%2FFacebook&code=AQBsffxl7L8x2FUvs4M_4R4rpckV3VoIGkWx2HWX8XxVOZbnvBwb7dXTHUs9UfS2CqAWRS5KGSxw5CEvqQcsCxZydGOKo9WEX76JZ2Pppyc5BCy7uEWMGfWZ390znZCFJCk1fAYDP4CotAaccceq5MCerS98zGXTJUgvC1RPlEIXtU8Y3IB03IjdhfMKsSFZodw7dotHDmGnDQ61KWbvQ6R5OPkuH3mKh7-vHJkvPLkDOypX4se97k5cokD6G4GOIvBnXM7qEdV_WQJmr6su1-Ng9jlF8ylHDUIdxEmDZZMm3Fctg8r_cRegLAN0VSfpfq6_IEzNTZ8i-iBdjx0CjxA_&client_secret=5bc95376dbe4effb7a5433ea876b2e50&code=AQBsffxl7L8x2FUvs4M_4R4rpckV3VoIGkWx2HWX8XxVOZbnvBwb7dXTHUs9UfS2CqAWRS5KGSxw5CEvqQcsCxZydGOKo9WEX76JZ2Pppyc5BCy7uEWMGfWZ390znZCFJCk1fAYDP4CotAaccceq5MCerS98zGXTJUgvC1RPlEIXtU8Y3IB03IjdhfMKsSFZodw7dotHDmGnDQ61KWbvQ6R5OPkuH3mKh7-vHJkvPLkDOypX4se97k5cokD6G4GOIvBnXM7qEdV_WQJmr6su1-Ng9jlF8ylHDUIdxEmDZZMm3Fctg8r_cRegLAN0VSfpfq6_IEzNTZ8i-iBdjx0CjxA_
*/
