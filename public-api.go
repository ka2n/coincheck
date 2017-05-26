package coincheck

import "context"

type TickerResponse struct {
	Last      float64 `json:"last"`
	Bid       float64 `json:"bid"`
	Ask       float64 `json:"ask"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"volume,string"`
	Timestamp int     `json:"timestamp"`
}

// Ticker returns latest ticker
func (c *Client) Ticker(ctx context.Context) (*TickerResponse, error) {
	req, err := c.newPublicRequest("GET", "ticker", nil, nil)
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

	var ret TickerResponse
	return &ret, c.decodeResponse(res, &ret, nil)
}
