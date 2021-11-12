package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp-demoapp/product-api-go/data"
	"github.com/hashicorp-demoapp/product-api-go/data/model"
	"github.com/hashicorp/go-hclog"
	"go.opentelemetry.io/otel/trace"
)

// Coffee -
type Coffee struct {
	con data.Connection
	log hclog.Logger
}

// NewCoffee
func NewCoffee(con data.Connection, l hclog.Logger) *Coffee {
	return &Coffee{con, l}
}

func (c *Coffee) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	log := c.log.With("http.scheme", r.URL.Scheme, "http.url", r.URL.String(), "http.host", r.Host, "http.user_agent", r.Header.Get("User-Agent"))
	log.Info("Handle Coffee")
	ctx := r.Context()
	spanCtx := trace.SpanContextFromContext(ctx)
	prods, err := c.con.GetProducts(ctx)
	if err != nil {
		c.log.Error("Unable to get products from database", "error", err)
		http.Error(rw, "Unable to list products", http.StatusInternalServerError)
	}

	d, err := prods.ToJSON()
	if err != nil {
		c.log.Error("Unable to convert products to JSON", "error", err)
		http.Error(rw, "Unable to list products", http.StatusInternalServerError)
	}

	rw.Write(d)
	t1 := time.Since(t0)
	log.With("span.id", spanCtx.SpanID(), "trace.id", spanCtx.TraceID()).Info("Handle Coffee Finish", "duration_ms", t1.Milliseconds())
}

// CreateCoffee creates a new coffee
func (c *Coffee) CreateCoffee(_ int, rw http.ResponseWriter, r *http.Request) {
	c.log.Info("Handle Coffee | CreateCoffee")
	ctx := r.Context()

	body := model.Coffee{}

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		c.log.Error("Unable to decode JSON", "error", err)
		http.Error(rw, "Unable to parse request body", http.StatusInternalServerError)
		return
	}

	coffee, err := c.con.CreateCoffee(ctx, body)
	if err != nil {
		c.log.Error("Unable to create new coffee", "error", err)
		http.Error(rw, fmt.Sprintf("Unable to create new coffee: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	d, err := coffee.ToJSON()
	if err != nil {
		c.log.Error("Unable to convert coffee to JSON", "error", err)
		http.Error(rw, "Unable to create new coffee", http.StatusInternalServerError)
	}

	rw.Write(d)
}
