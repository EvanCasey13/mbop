package keycloak

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/redhatinsights/mbop/internal/config"
	l "github.com/redhatinsights/mbop/internal/logger"
	"github.com/redhatinsights/mbop/internal/models"
)

type Client struct {
	client *http.Client
}

func (keyCloak *Client) InitKeycloakConnection() error {
	keyCloak.client = &http.Client{
		Timeout: time.Duration(config.Get().KeyCloakTimeout * int64(time.Second)),
	}

	return nil
}

func (keyCloak *Client) GetUsers(token string, u models.UserBody, q models.UserV1Query) (models.Users, error) {
	users := models.Users{Users: []models.User{}}
	url, err := createV1RequestURL(u, q)
	if err != nil {
		return users, err
	}

	body, err := keyCloak.sendKeycloakGetRequest(url, token)
	if err != nil {
		l.Log.Error(err, "/v3/users error sending request")
		return users, err
	}

	unmarshaledResponse := models.KeycloakResponses{}
	err = json.Unmarshal(body, &unmarshaledResponse)
	if err != nil {
		return users, err
	}

	return keycloakResponseToUsers(unmarshaledResponse.Users), err
}

func (keyCloak *Client) GetAccountV3Users(orgID string, token string, q models.UserV3Query) (models.Users, error) {
	users := models.Users{Users: []models.User{}}
	url, err := createV3UsersRequestURL(orgID, q)
	if err != nil {
		return users, err
	}

	body, err := keyCloak.sendKeycloakGetRequest(url, token)
	if err != nil {
		l.Log.Error(err, "/v3/users error sending request")
		return users, err
	}

	unmarshaledResponse := models.KeycloakResponses{}
	err = json.Unmarshal(body, &unmarshaledResponse)
	if err != nil {
		return users, err
	}

	return keycloakResponseToUsers(unmarshaledResponse.Users), nil
}

func (keyCloak *Client) GetAccessToken() (string, error) {
	token := models.KeycloakTokenObject{}
	url, err := createTokenURL()
	if err != nil {
		return "", err
	}

	body := createEncodedTokenBody()

	resp, err := http.Post(url.String(), "application/x-www-form-urlencoded", body)
	if err != nil {
		return "", fmt.Errorf("error fetching keycloak token response: %s", err)
	}

	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading keycloak token response body: %s", err)
	}

	err = json.Unmarshal(responseBody, &token)
	if err != nil {
		return "", fmt.Errorf("error unmarshling keycloak token response: %s", err)
	}

	return token.AccessToken, nil
}

func (keyCloak *Client) sendKeycloakGetRequest(url *url.URL, token string) ([]byte, error) {
	var responseBody []byte

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return responseBody, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := keyCloak.client.Do(req)
	if err != nil {
		l.Log.Error(err, "error fetching keycloak response")
		return responseBody, err
	}

	responseBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		l.Log.Error(err, "error reading keycloak response body")
		return responseBody, err
	}

	// Close response body
	defer resp.Body.Close()

	return responseBody, nil
}

func createEncodedTokenBody() *strings.Reader {
	data := url.Values{}
	data.Set("username", config.Get().KeyCloakTokenUsername)
	data.Set("password", config.Get().KeyCloakTokenPassword)
	data.Set("grant_type", config.Get().KeyCloakTokenGrantType)
	data.Set("client_id", config.Get().KeyCloakTokenClientID)

	return strings.NewReader(data.Encode())
}

func createTokenURL() (*url.URL, error) {
	url, err := url.Parse(fmt.Sprintf("%s://%s:%s/token", config.Get().KeyCloakProtocol, config.Get().KeyCloakHost, config.Get().KeyCloakPort))
	if err != nil {
		return nil, fmt.Errorf("error creating keycloak token url: %s", err)
	}

	return url, err
}

// MAKE response to users function to massage data back to regular format
func createV1RequestURL(usernames models.UserBody, q models.UserV1Query) (*url.URL, error) {
	url, err := url.Parse(fmt.Sprintf("%s://%s:%s/users?limit=100", config.Get().KeyCloakProtocol, config.Get().KeyCloakHost, config.Get().KeyCloakPort))
	if err != nil {
		return nil, fmt.Errorf("error creating (keycloak) /users url: %s", err)
	}

	queryParams := url.Query()

	if q.QueryBy != "" {
		queryParams.Add("order", q.QueryBy)
	}

	if q.SortOrder != "" {
		queryParams.Add("direction", q.SortOrder)
	}

	queryParams.Add("usernames", createUsernamesQuery(usernames.Users))

	url.RawQuery = queryParams.Encode()
	return url, err
}

func createV3UsersRequestURL(orgID string, q models.UserV3Query) (*url.URL, error) {
	url, err := url.Parse(fmt.Sprintf("%s://%s:%s/users", config.Get().KeyCloakProtocol, config.Get().KeyCloakHost, config.Get().KeyCloakPort))
	if err != nil {
		return nil, fmt.Errorf("error creating (keycloak) /v3/users url: %s", err)
	}
	queryParams := url.Query()

	if q.SortOrder != "" {
		queryParams.Add("direction", q.SortOrder)
	}

	queryParams.Add("org_id", orgID)
	queryParams.Add("limit", strconv.Itoa(q.Limit))
	queryParams.Add("offset", strconv.Itoa(q.Offset))

	url.RawQuery = queryParams.Encode()

	return url, err
}

func createUsernamesQuery(usernames []string) string {
	usernameQuery := ""

	for _, username := range usernames {
		if usernameQuery == "" {
			usernameQuery += username
		} else {
			usernameQuery += fmt.Sprintf(",%s", username)
		}
	}

	return usernameQuery
}

func keycloakResponseToUsers(r []models.KeycloakResponse) models.Users {
	users := models.Users{}

	for _, response := range r {
		users.AddUser(models.User{
			Username:      response.Username,
			ID:            response.ID,
			Email:         response.Email,
			FirstName:     response.FirstName,
			LastName:      response.LastName,
			AddressString: "",
			IsActive:      true,
			IsInternal:    response.IsInternal,
			Locale:        "en_US",
			OrgID:         response.OrgID,
			DisplayName:   response.UserID,
			Type:          response.Type,
		})
	}

	return users
}
