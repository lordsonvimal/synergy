package logger

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	otelTrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Logger interface for flexibility
type Logger interface {
	WithContext(ctx context.Context) Logger
	Info(ctx context.Context, msg string, fields map[string]any)
	Warn(ctx context.Context, msg string, fields map[string]any)
	Error(ctx context.Context, msg string, fields map[string]any)
	Fatal(ctx context.Context, msg string, fields map[string]any)
	Close()
}

// LogrusLogger implements the Logger interface
type LogrusLogger struct {
	contextEntry *logrus.Entry
	logger       *logrus.Logger
	tracer       otelTrace.Tracer
	logChannel   chan *logrus.Entry
	shutdown     chan struct{}
	sync.WaitGroup
}

var (
	instance Logger // Use the Logger interface
	once     sync.Once
)

func InitLogger(serviceName string) {
	once.Do(func() {
		log := logrus.New()
		log.SetFormatter(&logrus.JSONFormatter{})

		// Multi-output: file + console for development
		log.SetOutput(os.Stdout) // Print to console during development

		// Use lumberjack for rotating logs
		log.SetOutput(&lumberjack.Logger{
			Filename:   "./logs/app.log",
			MaxSize:    10,   // MB
			MaxBackups: 3,    // Maximum backup files
			MaxAge:     28,   // Retention period (days)
			Compress:   true, // Compress old logs
		})

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

		// Start log processing workers
		for i := 0; i < 4; i++ {
			instance.(*LogrusLogger).Add(1)
			go instance.(*LogrusLogger).processLogQueue()
		}

		if tp := initTracerProvider(); tp != nil {
			instance.(*LogrusLogger).tracer = tp.Tracer(serviceName)
			log.Info("OpenTelemetry tracing enabled")
		} else {
			log.Warn("OpenTelemetry tracing disabled")
		}
	})
}

// GetLogger returns the logger as the Logger interface
func GetLogger() Logger {
	return instance
}

// processLogQueue processes logs asynchronously
func (l *LogrusLogger) processLogQueue() {
	defer l.Done()
	for {
		select {
		case logEntry := <-l.logChannel:
			_, err := logEntry.Logger.Out.Write([]byte(logEntry.Message + "\n"))
			if err != nil {
				logrus.Errorf("Failed to write log: %v", err)
			}
		case <-l.shutdown:
			return
		}
	}
}

func (l *LogrusLogger) WithContext(ctx context.Context) Logger {
	// Extract trace and span IDs
	spanCtx := otelTrace.SpanContextFromContext(ctx)
	fields := map[string]any{}

	if spanCtx.HasTraceID() {
		fields["trace_id"] = spanCtx.TraceID().String()
		fields["span_id"] = spanCtx.SpanID().String()
	}

	// Create a new logger with contextual fields
	return &LogrusLogger{
		logger:       l.logger,                    // Base logger
		contextEntry: l.logger.WithFields(fields), // Contextual entry
		tracer:       l.tracer,
		logChannel:   l.logChannel,
		shutdown:     l.shutdown,
	}
}

func (l *LogrusLogger) asyncLog(ctx context.Context, level logrus.Level, msg string, fields map[string]any) {
	// Add trace and span IDs from the context
	spanCtx := otelTrace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		fields["trace_id"] = spanCtx.TraceID().String()
		fields["span_id"] = spanCtx.SpanID().String()
	}

	// Use contextEntry if available
	var entry *logrus.Entry
	if l.contextEntry != nil {
		entry = l.contextEntry.WithFields(fields)
	} else {
		entry = l.logger.WithFields(fields)
	}

	entry.Level = level
	entry.Message = msg
	l.logChannel <- entry
}

func (l *LogrusLogger) Info(ctx context.Context, msg string, fields map[string]any) {
	l.asyncLog(ctx, logrus.InfoLevel, msg, fields)
}

func (l *LogrusLogger) Warn(ctx context.Context, msg string, fields map[string]any) {
	l.asyncLog(ctx, logrus.WarnLevel, msg, fields)
}

func (l *LogrusLogger) Error(ctx context.Context, msg string, fields map[string]any) {
	l.asyncLog(ctx, logrus.ErrorLevel, msg, fields)
}

func (l *LogrusLogger) Fatal(ctx context.Context, msg string, fields map[string]any) {
	l.asyncLog(ctx, logrus.FatalLevel, msg, fields)
	os.Exit(1)
}

func (l *LogrusLogger) Close() {
	close(l.shutdown)
	l.Wait()
	close(l.logChannel)
}

// initTracerProvider initializes OpenTelemetry tracing
func initTracerProvider() *sdkTrace.TracerProvider {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		return nil
	}

	grpcExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelEndpoint),
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
