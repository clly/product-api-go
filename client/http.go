package client

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp-demoapp/product-api-go/data/model"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTP contains all client details
type HTTP struct {
	client  *http.Client
	baseURL string
}

// NewHTTP creates a new HTTP client
func NewHTTP(baseURL string) *HTTP {
	c := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	return &HTTP{c, baseURL}
}

// GetCoffees retrieves a list of coffees
func (h *HTTP) GetCoffees(ctx context.Context) ([]model.Coffee, error) {
	log.Print("INFO: Executing GetCoffees")
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/coffees", h.baseURL), nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}

	coffees := model.Coffees{}
	coffees.FromJSON(resp.Body)
	if err != nil {
		return nil, err
	}

	return coffees, nil
}

// GetCoffee retrieves a single coffee
func (h *HTTP) GetCoffee(coffeeID int) (*model.Coffee, error) {
	resp, err := h.client.Get(fmt.Sprintf("%s/coffees/%d", h.baseURL, coffeeID))
	if err != nil {
		return nil, err
	}

	coffee := model.Coffee{}
	err = coffee.FromJSON(resp.Body)
	if err != nil {
		return nil, err
	}

	return &coffee, nil
}

// GetIngredientsForCoffee retrieves a list of ingredients that go into a particular coffee
func (h *HTTP) GetIngredientsForCoffee(coffeeID int) ([]model.Ingredient, error) {
	resp, err := h.client.Get(fmt.Sprintf("%s/coffees/%d/ingredients", h.baseURL, coffeeID))
	if err != nil {
		return nil, err
	}

	ingredients := model.Ingredients{}
	err = ingredients.FromJSON(resp.Body)
	if err != nil {
		return nil, err
	}

	return ingredients, nil
}
