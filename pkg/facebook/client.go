package facebook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	dialogBaseURL  = "https://www.facebook.com"
	graphBaseURL   = "https://graph.facebook.com"
)

// Client talks to the Facebook OAuth dialog and Graph API.
type Client struct {
	AppID       string
	AppSecret   string
	RedirectURI string
	APIVersion  string
	HTTPClient  *http.Client
}

func NewClient(appID, appSecret, redirectURI, apiVersion string) *Client {
	return &Client{
		AppID:       appID,
		AppSecret:   appSecret,
		RedirectURI: redirectURI,
		APIVersion:  apiVersion,
		HTTPClient:  &http.Client{Timeout: 15 * time.Second},
	}
}

// AuthURL builds the OAuth dialog URL a tenant is redirected to.
func (c *Client) AuthURL(state string, scopes []string) string {
	params := url.Values{}
	params.Set("client_id", c.AppID)
	params.Set("redirect_uri", c.RedirectURI)
	params.Set("state", state)
	params.Set("response_type", "code")
	if len(scopes) > 0 {
		params.Set("scope", strings.Join(scopes, ","))
	}
	return dialogBaseURL + "/" + c.APIVersion + "/dialog/oauth?" + params.Encode()
}

// TokenResponse is the shape of Facebook's /oauth/access_token response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// ExchangeCode swaps the authorization code for a short-lived user token.
func (c *Client) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	params := url.Values{}
	params.Set("client_id", c.AppID)
	params.Set("client_secret", c.AppSecret)
	params.Set("redirect_uri", c.RedirectURI)
	params.Set("code", code)
	return c.getToken(ctx, params)
}

// ExchangeLongLived swaps a short-lived user token for a long-lived one
// (typically ~60 days for user tokens).
func (c *Client) ExchangeLongLived(ctx context.Context, shortLivedToken string) (*TokenResponse, error) {
	params := url.Values{}
	params.Set("grant_type", "fb_exchange_token")
	params.Set("client_id", c.AppID)
	params.Set("client_secret", c.AppSecret)
	params.Set("fb_exchange_token", shortLivedToken)
	return c.getToken(ctx, params)
}

func (c *Client) getToken(ctx context.Context, params url.Values) (*TokenResponse, error) {
	u := graphBaseURL + "/" + c.APIVersion + "/oauth/access_token?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("facebook token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("facebook token exchange returned %d: %s", resp.StatusCode, string(body))
	}

	var out TokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.AccessToken == "" {
		return nil, fmt.Errorf("facebook token exchange returned empty token: %s", string(body))
	}
	return &out, nil
}

// graphError captures a Facebook API error payload.
type graphError struct {
	Error struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error"`
}

func graphErr(status int, body []byte) error {
	var ge graphError
	if err := json.Unmarshal(body, &ge); err == nil && ge.Error.Message != "" {
		return fmt.Errorf("facebook api returned %d: %s (code %d)", status, ge.Error.Message, ge.Error.Code)
	}
	return fmt.Errorf("facebook api returned %d: %s", status, string(body))
}

// getJSON performs an authenticated Graph GET and decodes JSON into out.
func (c *Client) getJSON(ctx context.Context, path string, token string, params url.Values, out interface{}) error {
	if params == nil {
		params = url.Values{}
	}
	if token != "" {
		params.Set("access_token", token)
	}
	u := graphBaseURL + "/" + c.APIVersion + path + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("facebook request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return graphErr(resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return err
	}
	return nil
}

// Paging mirrors Graph API cursor paging.
type Paging struct {
	Cursors struct {
		Before string `json:"before"`
		After  string `json:"after"`
	} `json:"cursors"`
	Next string `json:"next"`
}

// AdAccount is a trimmed ad account object.
type AdAccount struct {
	ID            string `json:"id"` // format: act_<id>
	Name          string `json:"name"`
	AccountStatus int    `json:"account_status"`
	Currency      string `json:"currency"`
}

// GetAdAccounts lists ad accounts visible to the token.
func (c *Client) GetAdAccounts(ctx context.Context, accessToken string) ([]AdAccount, error) {
	params := url.Values{}
	params.Set("fields", "id,name,account_status,currency")
	params.Set("limit", "50")
	var out struct {
		Data []AdAccount `json:"data"`
	}
	if err := c.getJSON(ctx, "/me/adaccounts", accessToken, params, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// Campaign is a trimmed campaign object.
type Campaign struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	EffectiveStatus string `json:"effective_status"`
	Objective       string `json:"objective"`
}

// GetCampaigns lists campaigns in an ad account (act_<id> or bare id both accepted).
func (c *Client) GetCampaigns(ctx context.Context, accessToken, adAccountID string) ([]Campaign, error) {
	id := adAccountID
	if !strings.HasPrefix(id, "act_") {
		id = "act_" + id
	}
	params := url.Values{}
	params.Set("fields", "id,name,status,effective_status,objective")
	params.Set("limit", "200")
	var out struct {
		Data []Campaign `json:"data"`
	}
	if err := c.getJSON(ctx, "/"+id+"/campaigns", accessToken, params, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// AdWithCampaign resolves an ad to its campaign.
type AdWithCampaign struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Campaign struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"campaign"`
}

// GetAdCampaign resolves an ad id to its parent campaign.
func (c *Client) GetAdCampaign(ctx context.Context, accessToken, adID string) (*AdWithCampaign, error) {
	params := url.Values{}
	params.Set("fields", "id,name,campaign{id,name}")
	var out AdWithCampaign
	if err := c.getJSON(ctx, "/"+adID, accessToken, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// LeadField is one submitted lead-gen field.
type LeadField struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// LeadgenLead is the full lead object behind a leadgen_id.
type LeadgenLead struct {
	ID          string      `json:"id"`
	CreatedTime string      `json:"created_time"`
	AdID        string      `json:"ad_id"`
	FormID      string      `json:"form_id"`
	FieldData   []LeadField `json:"field_data"`
}

// GetLeadgenLead fetches the full lead payload for a leadgen_id.
func (c *Client) GetLeadgenLead(ctx context.Context, accessToken, leadgenID string) (*LeadgenLead, error) {
	params := url.Values{}
	params.Set("fields", "id,created_time,ad_id,form_id,field_data")
	var out LeadgenLead
	if err := c.getJSON(ctx, "/"+leadgenID, accessToken, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Page is a trimmed page object.
type Page struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetPages lists pages visible to the token (best-effort, used for webhook tenant resolution).
func (c *Client) GetPages(ctx context.Context, accessToken string) ([]Page, error) {
	params := url.Values{}
	params.Set("fields", "id,name")
	params.Set("limit", "50")
	var out struct {
		Data []Page `json:"data"`
	}
	if err := c.getJSON(ctx, "/me/accounts", accessToken, params, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// MeResponse is a trimmed /me response.
type MeResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetMe returns the id/name of the user for the given token.
func (c *Client) GetMe(ctx context.Context, accessToken string) (*MeResponse, error) {
	params := url.Values{}
	params.Set("fields", "id,name")
	var out MeResponse
	if err := c.getJSON(ctx, "/me", accessToken, params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
