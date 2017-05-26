package coincheck

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"
)

func New(key string, secret string) (*Client, error) {
	u, err := url.Parse("https://coincheck.com/api")
	if err != nil {
		return nil, err
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)

	return &Client{
		Key:        key,
		Secret:     secret,
		BaseURL:    u,
		HTTPClient: http.DefaultClient,
		Logger:     logger,
	}, nil
}

type Client struct {
	Key        string
	Secret     string
	BaseURL    *url.URL
	HTTPClient *http.Client
	Logger     *log.Logger
}

type APIResponse struct {
	Success bool
	Error   string
}

type PaginationResponse struct {
	Limit         int64
	Order         string
	StartingAfter int64
	EndingBefore  int64
}

type PaginationRequest PaginationResponse

func (p *PaginationRequest) AddValues(v url.Values) {
	if p == nil {
		return
	}
	if p.Limit > 0 {
		v.Add("limit", strconv.FormatInt(p.Limit, 10))
	}
	if p.Order != "" {
		v.Add("order", p.Order)
	}
	if p.StartingAfter != 0 {
		v.Add("starting_after", strconv.FormatInt(p.StartingAfter, 10))
	}
	if p.EndingBefore != 0 {
		v.Add("ending_before", strconv.FormatInt(p.EndingBefore, 10))
	}
}

type OrderHistory struct {
	ID          int               `json:"id"`
	OrderID     int               `json:"order_id"`
	CreatedAt   time.Time         `json:"created_at"`
	Funds       map[string]string `json:"funds"`
	Pair        string            `json:"pair"`
	Rate        string            `json:"rate"`
	FeeCurrency string            `json:"fee_currency"`
	Fee         string            `json:"fee"`
	Liquidity   string            `json:"liquidity"`
	Side        string            `json:"side"`
}

type OrderHistoryResponse struct {
	PaginationResponse
	APIResponse
	Data         []OrderHistory
	Transactions []OrderHistory
}

func (c *Client) OrderHistory(ctx context.Context, page *PaginationRequest) (*OrderHistoryResponse, error) {
	param := url.Values{}
	page.AddValues(param)

	var endpoint string
	if page != nil {
		endpoint = "/exchange/orders/transactions_pagination"
	} else {
		endpoint = "/exchange/orders/transactions"
	}

	req, err := c.newRequest("GET", endpoint, param, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	c.Logger.Println("GET", req.URL.String())
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var ret OrderHistoryResponse
	// f, _ := os.Create("order_history.json")
	if err := c.decodeResponse(res, &ret, nil); err != nil {
		return nil, err
	}

	if ret.Error != "" {
		return nil, errors.New(ret.Error)
	}
	if !ret.Success {
		return nil, errors.New("unkonwn API error")
	}

	return &ret, nil
}

type SentHistory struct {
	ID        int       `json:"id"`
	Amount    float64   `json:"amount,string"`
	Currency  string    `json:"currency"`
	Fee       float64   `json:"fee,string"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
}

type DepositHistory struct {
	ID          int       `json:"id"`
	Amount      float64   `json:"amount,string"`
	Currency    string    `json:"currency"`
	Address     string    `json:"address"`
	Status      string    `json:"status"`
	ConfirmedAt time.Time `json:"confirmed_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type SentHistoryResponse struct {
	PaginationResponse
	APIResponse
	Sends []SentHistory
}

func (c *Client) SentHistory(ctx context.Context, currency string) (*SentHistoryResponse, error) {
	param := url.Values{"currency": []string{currency}}

	req, err := c.newRequest("GET", "/send_money", param, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	c.Logger.Println("GET", req.URL.String())
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var ret SentHistoryResponse
	if err := c.decodeResponse(res, &ret, nil); err != nil {
		return nil, err
	}

	if ret.Error != "" {
		return nil, errors.New(ret.Error)
	}
	if !ret.Success {
		return nil, errors.New("unkonwn API error")
	}

	return &ret, nil
}

type DepositHistoryResponse struct {
	PaginationResponse
	APIResponse
	Deposits []DepositHistory
}

func (c *Client) DepositHistory(ctx context.Context, currency string) (*DepositHistoryResponse, error) {
	param := url.Values{"currency": []string{currency}}

	req, err := c.newRequest("GET", "/deposit_money", param, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	c.Logger.Println("GET", req.URL.String())
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var ret DepositHistoryResponse
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&ret); err != nil {
		return nil, err
	}

	if ret.Error != "" {
		return nil, errors.New(ret.Error)
	}

	if !ret.Success {
		return nil, errors.New("unknown API error")
	}

	return &ret, nil
}

func (c *Client) OpenOrders(ctx context.Context) error {
	req, err := c.newRequest("GET", "/exchange/orders/opens", nil, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	b, _ := ioutil.ReadAll(res.Body)
	fmt.Println(string(b))

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %d", res.StatusCode)
	}

	return nil
}

func makeSign(nonce string, urlStr string, body []byte, secret string) (string, error) {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(nonce))
	h.Write([]byte(urlStr))
	h.Write([]byte(body))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (c *Client) newPublicRequest(method string, pathStr string, query url.Values, body io.Reader) (*http.Request, error) {
	u := *c.BaseURL
	u.Path = path.Join(c.BaseURL.Path, pathStr)
	if query != nil {
		u.RawQuery = query.Encode()
	}
	reqURL := u.String()
	return http.NewRequest(method, reqURL, body)
}

func (c *Client) newRequest(method string, pathStr string, query url.Values, body io.Reader) (*http.Request, error) {
	u := *c.BaseURL
	u.Path = path.Join(c.BaseURL.Path, pathStr)
	if query != nil {
		u.RawQuery = query.Encode()
	}
	reqURL := u.String()

	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)

	bb := bytes.NewBuffer(nil)
	if body != nil {
		body = io.TeeReader(body, bb)
	}
	sign, err := makeSign(nonce, reqURL, bb.Bytes(), c.Secret)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("ACCESS-KEY", c.Key)
	req.Header.Add("ACCESS-NONCE", nonce)
	req.Header.Add("ACCESS-SIGNATURE", sign)

	return req, nil
}

func (c *Client) decodeResponse(resp *http.Response, result interface{}, f *os.File) error {
	defer resp.Body.Close()

	if f != nil {
		resp.Body = ioutil.NopCloser(io.TeeReader(resp.Body, f))
		defer f.Close()
	}

	decoder := json.NewDecoder(resp.Body)
	return decoder.Decode(result)
}
