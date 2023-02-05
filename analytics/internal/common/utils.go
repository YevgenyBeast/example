package common

import (
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// InitConfig инициализация конфигурационного файла
func InitConfig(path string) error {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}

// InitLogger инициализация логгера
func InitLogger() *logrus.Entry {
	l := logrus.New()
	l.SetFormatter(new(logrus.JSONFormatter))
	logger := logrus.NewEntry(l).WithFields(logrus.Fields{"app": "analytics", "traceID": uuid.NewV4()})
	return logger
}

// InitOtel инициализация подключения к jaeger
func InitOtel() error {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(
		jaeger.WithEndpoint(viper.GetString("jaeger-collector")),
	))
	if err != nil {
		return err
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(viper.GetString("service-name")),
		)),
	)

	otel.SetTracerProvider(tp)

	return nil
}
