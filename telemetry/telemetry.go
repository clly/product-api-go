package telemetry

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"go.opentelemetry.io/otel"

	//"go.opentelemetry.io/otel/exporter/metric/stdout"
	otlpgrpc "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const SERVICE_NAME = "product-api-go"

/*type Telemetry struct {
	pusher   *push.Controller
	meter    api.Meter
	measures map[string]*api.Float64Measure
	counters map[string]*api.Float64Counter
}

func New(bind_address string) *Telemetry {
	selector := simple.NewWithExactMeasure()
	exporter, err := prometheus.NewExporter(prometheus.Options{})

	if err != nil {
		log.Panicf("failed to initialize metric stdout exporter %v", err)
	}

	batcher := defaultkeys.New(selector, sdkmetric.NewDefaultLabelEncoder(), false)
	pusher := push.New(batcher, exporter, time.Second)
	pusher.Start()

	go func() {
		_ = http.ListenAndServe(bind_address, exporter)
	}()

	global.SetMeterProvider(pusher)

	meter := global.MeterProvider().Meter("ex.com/basic")

	m := make(map[string]*api.Float64Measure)
	c := make(map[string]*api.Float64Counter)

	return &Telemetry{pusher, meter, m, c}
}

// AddMeasure to metrics
func (t *Telemetry) AddMeasure(key string) {
	met := t.meter.NewFloat64Measure(key)
	t.measures[key] = &met
}

// AddCounter to metrics collection
func (t *Telemetry) AddCounter(key string) {
	met := t.meter.NewFloat64Counter(key)
	t.counters[key] = &met
}

// NewTiming creates a new timing metric and returns a done function
func (t *Telemetry) NewTiming(key string) func() {
	// record the start time
	st := time.Now()

	return func() {
		dur := time.Now().Sub(st).Nanoseconds()
		handler := t.measures[key].AcquireHandle(nil)
		defer handler.Release()

		t.meter.RecordBatch(
			context.Background(),
			nil,
			t.measures[key].Measurement(float64(dur)),
		)
	}
}*/

func InitTracer() (context.Context, func(), error) {
	ctx := context.Background()

	//otel.SetErrorHandler()

	var exporter sdktrace.SpanExporter // allows overwrite in --test mode
	var err error

	exporter, err = otlpgrpc.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to configure OTLP/GRPC exporter: %s", err)
	}

	// set the service name that will show up in tracing UIs
	resAttrs := resource.WithAttributes(semconv.ServiceNameKey.String(SERVICE_NAME))
	res, err := resource.New(ctx, resAttrs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OpenTelemetry service name resource: %s", err)
	}

	// SSP sends all completed spans to the exporter immediately and that is
	// exactly what we want/need in this app
	// https://github.com/open-telemetry/opentelemetry-go/blob/main/sdk/trace/simple_span_processor.go
	//exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	ssp := sdktrace.NewBatchSpanProcessor(exporter)

	// ParentBased/AlwaysSample Sampler is the default and that's fine for this
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(ssp),
	)

	// inject the tracer into the otel globals (and this starts the background stuff, I think)
	otel.SetTracerProvider(tracerProvider)

	// set up the W3C trace context as the global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// callers need to defer this to make sure all the data gets flushed out
	return ctx, func() {
		err = tracerProvider.Shutdown(ctx)
		if err != nil {
			hclog.Default().Error("shutdown of OpenTelemetry tracerProvider failed: %s", err)
		}

		err = exporter.Shutdown(ctx)
		if err != nil {
			hclog.Default().Error("shutdown of OpenTelemetry OTLP exporter failed: %s", err)
		}
	}, nil
}
