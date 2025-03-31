package logger

import (
	"context"
	"os"
	"sync"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
	otelTrace "go.opentelemetry.io/otel/trace"
)

type loggerKey string

const LoggerKey loggerKey = "logger"

type Logger interface {
	Flush()
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

type LoggerConfig struct {
	Environment string
	LogLevel    string
	LogFile     string
}

var (
	instance Logger
	once     sync.Once
)

func InitLogger(serviceName string) {
	once.Do(func() {
		log := logrus.New()
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
		log.SetOutput(os.Stdout)
		log.SetLevel(logrus.InfoLevel)

		instance = &LogrusLogger{
			logger:     log,
			logChannel: make(chan *logrus.Entry, 1000),
			shutdown:   make(chan struct{}),
		}

		// Start log queue workers
		for range 4 {
			instance.(*LogrusLogger).Add(1)
			go instance.(*LogrusLogger).processLogQueue()
		}
	})
}

func ConfigureLogger(ctx context.Context, cfg *LoggerConfig) {
	log := GetLogger().(*LogrusLogger).logger // Get the existing logger

	// Apply new configurations
	if cfg.Environment == "production" {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	}

	if cfg.LogFile != "" {
		log.SetOutput(&lumberjack.Logger{
			Filename:   cfg.LogFile,
			MaxSize:    10, // MB
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		})
	} else {
		log.SetOutput(os.Stdout)
	}

	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)
}

func GetLogger() Logger {
	if instance == nil {
		panic("Logger not initialized. Call InitLogger() first.")
	}
	return instance
}

func (l *LogrusLogger) processLogQueue() {
	defer l.Done()
	for {
		select {
		case logEntry := <-l.logChannel:
			logEntry.Log(logEntry.Level, logEntry.Message)
		case <-l.shutdown:
			return
		}
	}
}

func (l *LogrusLogger) Flush() {
	close(l.shutdown) // Signal the log workers to stop
	l.Wait()          // Wait for all logs to be processed
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
		logger:       l.logger,
		contextEntry: l.logger.WithFields(fields),
		tracer:       l.tracer,
		logChannel:   l.logChannel,
		shutdown:     l.shutdown,
	}
}

func (l *LogrusLogger) asyncLog(ctx context.Context, level logrus.Level, msg string, fields map[string]any) {
	spanCtx := otelTrace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		fields["trace_id"] = spanCtx.TraceID().String()
		fields["span_id"] = spanCtx.SpanID().String()
	}

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
