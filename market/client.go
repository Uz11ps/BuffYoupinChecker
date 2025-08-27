package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"
)

type Client struct {
	apiKey  string
	baseURL string
	limiter *rate.Limiter
	client  *http.Client
}

type PriceResponse struct {
	Success  bool        `json:"success"`
	Time     int64       `json:"time"`
	Currency string      `json:"currency"`
	Items    []PriceItem `json:"items"`
}

type PriceItem struct {
	MarketHashName string `json:"market_hash_name"`
	Volume         string `json:"volume"`
	Price          string `json:"price"`
}

type ItemInfo struct {
	Success bool     `json:"success"`
	History []string `json:"history"`
}

func NewClient(apiKey string) *Client {
	// Устанавливаем лимит 4 запроса в секунду (меньше 5 для безопасности)
	limiter := rate.NewLimiter(rate.Limit(4), 1)
	
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://market.csgo.com/api/v2",
		limiter: limiter,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) makeRequest(endpoint string, params url.Values) ([]byte, error) {
	// Ждем разрешения от rate limiter
	ctx := context.Background()
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Добавляем обязательные параметры
	params.Set("key", c.apiKey)
	params.Set("v", "2")

	reqURL := fmt.Sprintf("%s/%s?%s", c.baseURL, endpoint, params.Encode())
	
	resp, err := c.client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	return body, nil
}

// Получение списка цен (лучшие предложения)  
func (c *Client) GetPrices() (*PriceResponse, error) {
	params := url.Values{}
	
	body, err := c.makeRequest("prices/RUB.json", params)
	if err != nil {
		return nil, err
	}

	var response PriceResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	return &response, nil
}

// Получение истории продаж по hash name
func (c *Client) GetItemHistory(hashName string) (*ItemInfo, error) {
	params := url.Values{}
	params.Set("hash_name", hashName)
	
	body, err := c.makeRequest("get-list-items-info", params)
	if err != nil {
		return nil, err
	}

	var response ItemInfo
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	return &response, nil
}

// Поиск предмета по hash name
func (c *Client) SearchItemByHashName(hashName string) ([]PriceItem, error) {
	params := url.Values{}
	params.Set("hash_name", hashName)
	
	body, err := c.makeRequest("search-item-by-hash-name", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Success bool        `json:"success"`
		Data    []PriceItem `json:"data"`
	}
	
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("API request failed")
	}

	return response.Data, nil
}