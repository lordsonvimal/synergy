package logger

import (
	"context"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	otelTrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
)

// Logger interface for flexibility
type Logger interface {
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Error(msg string, fields map[string]interface{})
	Fatal(msg string, fields map[string]interface{})
}

// LogrusLogger implements the Logger interface
type LogrusLogger struct {
	logger     *logrus.Logger
	tracer     otelTrace.Tracer
	logChannel chan *logrus.Entry
	shutdown   chan struct{}
	sync.WaitGroup
}

var (
	instance *LogrusLogger
	once     sync.Once
)

// InitLogger initializes Logrus and OpenTelemetry tracing
func InitLogger(serviceName string) {
	once.Do(func() {
		log := logrus.New()
		log.SetFormatter(&logrus.JSONFormatter{})
		log.SetOutput(os.Stdout)

		level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
		if err != nil {
			level = logrus.InfoLevel
		}
		log.SetLevel(level)

		instance = &LogrusLogger{
			logger:     log,
			tracer:     noop.NewTracerProvider().Tracer("noop"),
			logChannel: make(chan *logrus.Entry, 1000),
			shutdown:   make(chan struct{}),
		}

		// Start async log processing workers
		for i := 0; i < 4; i++ {
			instance.Add(1)
			go instance.processLogQueue()
		}

		// Initialize OpenTelemetry tracing
		if tp := initTracerProvider(serviceName); tp != nil {
			instance.tracer = tp.Tracer(serviceName)
			instance.logger.Info("OpenTelemetry tracing enabled")
		} else {
			instance.logger.Warn("OpenTelemetry tracing disabled")
		}
	})
}

// GetLogger returns the logger instance
func GetLogger() Logger {
	return instance
}

// processLogQueue processes logs asynchronously
func (l *LogrusLogger) processLogQueue() {
	defer l.Done()
	for {
		select {
		case logEntry := <-l.logChannel:
			logEntry.Logger.Out.Write([]byte(logEntry.Message + "\n"))
		case <-l.shutdown:
			return
		}
	}
}

// AsyncLog sends log messages to the queue
func (l *LogrusLogger) asyncLog(level logrus.Level, msg string, fields map[string]interface{}) {
	entry := l.logger.WithFields(fields)
	entry.Level = level
	entry.Message = msg
	l.logChannel <- entry
}

// Info logs an info message
func (l *LogrusLogger) Info(msg string, fields map[string]interface{}) {
	l.asyncLog(logrus.InfoLevel, msg, fields)
}

// Warn logs a warning message
func (l *LogrusLogger) Warn(msg string, fields map[string]interface{}) {
	l.asyncLog(logrus.WarnLevel, msg, fields)
}

// Error logs an error message
func (l *LogrusLogger) Error(msg string, fields map[string]interface{}) {
	l.asyncLog(logrus.ErrorLevel, msg, fields)
}

// Fatal logs a fatal error and exits
func (l *LogrusLogger) Fatal(msg string, fields map[string]interface{}) {
	l.asyncLog(logrus.FatalLevel, msg, fields)
	os.Exit(1)
}

// CloseLogger ensures all logs are processed before shutdown
func CloseLogger() {
	close(instance.shutdown)
	instance.Wait()
	close(instance.logChannel)
}

// initTracerProvider initializes OpenTelemetry tracing
func initTracerProvider(serviceName string) *sdkTrace.TracerProvider {
	ctx := context.Background()
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		return nil
	}

	grpcExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelEndpoint),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil {
		return nil
	}

	tp := sdkTrace.NewTracerProvider(
		sdkTrace.WithBatcher(grpcExporter),
		sdkTrace.WithResource(resource.Default()),
	)

	otel.SetTracerProvider(tp)
	return tp
}
