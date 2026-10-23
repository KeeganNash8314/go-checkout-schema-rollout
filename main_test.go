package main

import "testing"

func TestCheckoutFlagChoosesFulfillmentAndReceipt(t *testing.T) {
	order := Order{ID: "ord-7", Total: 4200, Address: "42 Market St"}
	tests := []struct {
		name                         string
		v2                           bool
		fulfillment, receipt, schema string
	}{
		{"legacy", false, "standard", "receipt-ord-7", "v1"},
		{"new path", true, "priority", "receipt-v2-ord-7", "v2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkout(order, tt.v2)
			if got.Fulfillment != tt.fulfillment || got.Receipt != tt.receipt || got.Schema != tt.schema {
				t.Fatalf("got %+v", got)
			}
		})
	}
}
