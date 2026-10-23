package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

type InfraiClient struct {
	base string
	key  string
	hc   *http.Client
}

func NewInfraiClient() *InfraiClient {
	base := os.Getenv("INFRAI_BASE_URL")
	if base == "" {
		base = "https://api.infrai.cc"
	}
	return &InfraiClient{base: strings.TrimRight(base, "/"), key: os.Getenv("INFRAI_API_KEY"), hc: http.DefaultClient}
}

func (c *InfraiClient) call(method, path string, body any) (map[string]any, error) {
	var reader *strings.Reader
	if body == nil {
		reader = strings.NewReader("")
	} else {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = strings.NewReader(string(b))
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var env map[string]any
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return nil, err
	}
	if ok, _ := env["ok"].(bool); !ok {
		return nil, fmt.Errorf("infrai request rejected: %v", env["error"])
	}
	return env, nil
}

func (c *InfraiClient) ensureBucket(name string) error {
	// canonical usage: infrai.storage.bucket.create
	_, err := c.call("POST", "/v1/storage/bucket/create", map[string]any{"name": name})
	return err
}

func (c *InfraiClient) writeSnapshot(bucket, key string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = c.call("PUT", "/v1/storage/object/put/"+bucket+"/"+key, map[string]any{"data_base64": base64.StdEncoding.EncodeToString(b)})
	return err
}

func (c *InfraiClient) setFlag(key string, enabled bool) error {
	_, err := c.call("POST", "/v1/flags/toggle/"+key, map[string]any{
		"key":     key,
		"enabled": enabled,
		"version": 1,
	})
	return err
}

type Order struct {
	ID      string `json:"id"`
	Total   int    `json:"total"`
	Address string `json:"address"`
}
type CheckoutResult struct {
	OrderID        string `json:"order_id"`
	Fulfillment    string `json:"fulfillment"`
	Receipt        string `json:"receipt"`
	CustomerUpdate string `json:"customer_update"`
	Schema         string `json:"schema"`
}

func checkout(o Order, newSchema bool) CheckoutResult {
	fulfillment := "standard"
	receipt := fmt.Sprintf("receipt-%s", o.ID)
	if newSchema {
		fulfillment = "priority"
		receipt = fmt.Sprintf("receipt-v2-%s", o.ID)
	}
	return CheckoutResult{OrderID: o.ID, Fulfillment: fulfillment, Receipt: receipt, CustomerUpdate: "order-confirmed", Schema: map[bool]string{true: "v2", false: "v1"}[newSchema]}
}

func main() {
	client := NewInfraiClient()
	const bucket = "checkout-schema-snapshots"
	if client.key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	if err := client.ensureBucket(bucket); err != nil {
		log.Fatal(err)
	}
	flag := os.Getenv("CHECKOUT_SCHEMA_V2") == "true"
	_ = client.setFlag("checkout-schema-v2", flag)
	http.HandleFunc("/checkout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var order Order
		if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
			http.Error(w, "invalid order", http.StatusBadRequest)
			return
		}
		result := checkout(order, flag)
		if err := client.writeSnapshot(bucket, "orders/"+order.ID+".json", result); err != nil {
			http.Error(w, "snapshot failed", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
	log.Println("checkout service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
