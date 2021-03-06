/*
Package ikaring provides http client Api for SplatNet; web service for Splatoon by Nintendo.
*/
package ikaring

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/bitly/go-simplejson"
)

// IkaClient is a http client for SplatNet.
// it includes http.Client.
type IkaClient struct {
	http.Client
}

const (
	splatoonCookieName = "_wag_session"
	splatoonDomainURL  = "https://splatoon.nintendo.net/"

	splatoonOauthURL = "https://splatoon.nintendo.net/users/auth/nintendo"
	nintendoOauthURL = "https://id.nintendo.net/oauth/authorize"

	splatoonScheduleAPI   = "https://splatoon.nintendo.net/schedule.json"
	splatoonRankingAPI    = "https://splatoon.nintendo.net/ranking.json"
	splatoonFriendListAPI = "https://splatoon.nintendo.net/friend_list/index.json"
)

// CreateClient generates ikaClient, http client object for Splatnet.
// It provides a http client with empty cookiejar.
func CreateClient() (*IkaClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := &IkaClient{}
	client.Jar = jar
	return client, nil
}

// SetSession sets session cookie to receiver IkaClient.
func (c *IkaClient) SetSession(session string) {
	uri, _ := url.Parse(splatoonDomainURL)
	c.Jar.SetCookies(uri, []*http.Cookie{
		&http.Cookie{
			Secure:   true,
			HttpOnly: true,
			Name:     splatoonCookieName,
			Value:    session,
		}})
}

// Login sends http request to authorize Nintendo Network.
// it require NNID and password and return session cookie.
func (c *IkaClient) Login(name string, password string) (string, error) {
	query, err := getOauthQuery(splatoonOauthURL, name, password)
	if err != nil {
		return "", err
	}

	resp, err := c.PostForm(nintendoOauthURL, query)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New(resp.Status)
	}

	session := getSessionFromCookie(resp.Cookies())

	return session, nil
}

// Authorized judges wheather the client authorized
// It checks cookies for session that used for authorization
func (c *IkaClient) Authorized() bool {
	uri, _ := url.Parse(splatoonDomainURL)
	session := getSessionFromCookie(c.Jar.Cookies(uri))
	return len(session) != 0
}

// GetStageInfo get Stage Info from SplatNet.
// this API send GET request and parse stage schedules from JSON.
func (c *IkaClient) GetStageInfo() (*StageInfo, error) {

	resp, err := c.Get(splatoonScheduleAPI)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err = checkJSONError(body); err != nil {
		return nil, err
	}

	return decodeJSONSchedule(body)
}

// GetRanking get Ranking of Friends from SplatNet.
// this API send GET request and parse ranking from JSON.
func (c *IkaClient) GetRanking() (*RankingInfo, error) {
	resp, err := c.Get(splatoonRankingAPI)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err = checkJSONError(body); err != nil {
		return nil, err
	}

	return decodeJSONRanking(body)
}

// GetFriendList get Friend List form SplatNet.
// this API send GET request and parse friend online status from JSON
func (c *IkaClient) GetFriendList() ([]Friend, error) {
	resp, err := c.Get(splatoonFriendListAPI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err = checkJSONError(body); err != nil {
		return nil, err
	}
	return decodeJSONFriendList(body)
}

// GetWeaponMap get Weapon Set from SplatNet.
// this API send GET request and parse weapon map by scraping HTML
func (c *IkaClient) GetWeaponMap() (map[string]string, error) {
	resp, err := c.Get(splatoonDomainURL)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, err
	}

	weapons := map[string]string{}
	doc.Find("#user_intention_weapon").Children().Each(func(_ int, s *goquery.Selection) {
		key, ok := s.Attr("value")
		if ok {
			weapons[key] = s.Text()
		}
	})
	return weapons, nil
}

func checkJSONError(data []byte) error {
	js, err := simplejson.NewJson(data)
	if err != nil {
		return err
	}

	if info := js.Get("error").MustString(); len(info) != 0 {
		return errors.New(info)
	}
	return nil
}
