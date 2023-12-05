package main

import (
	"context"
	"net/http"
	"os"

	"github.com/VictoriaMetrics/metrics"
	"github.com/gonzolino/gotado/v2"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var (
	logger    *logrus.Logger
	environ   EnvSpec
	tadoZones []*gotado.Zone
	ctx       context.Context
)

const (
	clientID     = "tado-web-app"
	clientSecret = "wZaRN7rpjn3FoNyF5IFuxg9uMzYJcvOoQ8QWiIqS3hfk6gLhVlG57j5YNoZL2Rtc"
)

type EnvSpec struct {
	UserName    string `envconfig:"USER_NAME" default:""`
	Password    string `envconfig:"PASSWORD" default:""`
	HomeName    string `envconfig:"HOME_NAME" default:""`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"debug"`
	MetricsPath string `envconfig:"METRICS_PATH" default:"/metrics"`
	ListenAddr  string `envconfig:"LISTEN_ADDR" default:":9783"`
}

func init() {
	err := envconfig.Process("", &environ)
	if err != nil {
		logger.Fatal("cannot parse env variables, err = ", err)
	}

	l, err := logrus.ParseLevel(environ.LogLevel)
	if err != nil {
		l = logrus.DebugLevel
	}
	logger = &logrus.Logger{
		Out:   os.Stderr,
		Level: l,
		Formatter: &easy.Formatter{
			TimestampFormat: "2006-01-02 15:04:05.000000",
			LogFormat:       "[%time%] %lvl% - %msg%\n",
		},
	}
}

func collect() {
	if tadoZones != nil {
		for _, zone := range tadoZones {
			logger.Debug("getting state for zone: ", zone.Name)
			if zoneState, err := zone.GetState(ctx); err != nil {
				logger.Fatal(err)
			} else {
				if zoneState == nil {
					logger.Fatal("got zoneState == nil!")
				} else {
					metrics.GetOrCreateFloatCounter(`tado_home_zone_temperature_celcius_degrees{zone_name="` + zone.Name + `"}`).Set(zoneState.SensorDataPoints.InsideTemperature.Celsius)
					metrics.GetOrCreateFloatCounter(`tado_home_zone_humidity_percentage{zone_name="` + zone.Name + `"}`).Set(zoneState.SensorDataPoints.Humidity.Percentage)
					metrics.GetOrCreateCounter(`tado_home_zone_timestamp_millis{zone_name="` + zone.Name + `"}`).Set(uint64(zoneState.SensorDataPoints.Humidity.Timestamp.UnixMilli()))
					metrics.GetOrCreateFloatCounter(`tado_home_zone_heating_power_percentage{zone_name="` + zone.Name + `"}`).Set(zoneState.ActivityDataPoints.HeatingPower.Percentage)
					metrics.GetOrCreateCounter(`tado_home_zone_heating_power_timestamp_millis{zone_name="` + zone.Name + `"}`).Set(uint64(zoneState.ActivityDataPoints.HeatingPower.Timestamp.UnixMilli()))
				}
				// fmt.Printf("%+v\n", zoneState.SensorDataPoints.InsideTemperature)
				// fmt.Printf("temperature: \t%s -> \t%s -> %f\n", zone.Name, zoneState.SensorDataPoints.InsideTemperature.Timestamp, zoneState.SensorDataPoints.InsideTemperature.Celsius)
				// fmt.Printf("humidity: \t%s -> \t%s -> %f\n", zone.Name, zoneState.SensorDataPoints.Humidity.Timestamp, zoneState.SensorDataPoints.Humidity.Percentage)
			}
		}
	}
}

func main() {
	ctx = context.Background()

	tado := gotado.New(clientID, clientSecret)
	logger.Info("logging in to tado.com by user ", environ.UserName)
	user, err := tado.Me(ctx, environ.UserName, environ.Password)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info("looking for home with name ", environ.HomeName)
	home, err := user.GetHome(ctx, environ.HomeName)

	logger.Info("getting home zones...")
	tadoZones, err = home.GetZones(ctx)
	if err != nil {
		logger.Fatal(err)
	}
	mux := http.NewServeMux()

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		collect()
		metrics.WritePrometheus(w, true)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
<head><title>Tado Exporter</title></head>
<body>
<h1>Idex Exporter</h1>
<p><a href="` + environ.MetricsPath + `">Metrics</a></p>
</body>
</html>`))
	})

	logger.Warn("exporter initialized!")
	_ = http.ListenAndServe(environ.ListenAddr, mux)
}
