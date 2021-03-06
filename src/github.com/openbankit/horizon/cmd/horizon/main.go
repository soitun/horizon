package main

import (
	"log"
	"runtime"

	"github.com/openbankit/go-base/amount"
	"github.com/openbankit/horizon"
	conf "github.com/openbankit/horizon/config"
	hlog "github.com/openbankit/horizon/log"
	"github.com/PuerkitoBio/throttled"
	"github.com/Sirupsen/logrus"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"time"
)

var app *horizon.App
var config conf.Config
var version string

var rootCmd *cobra.Command

func main() {
	if version != "" {
		horizon.SetVersion(version)
	}
	runtime.GOMAXPROCS(runtime.NumCPU())
	rootCmd.Execute()
}

func init() {
	godotenv.Load()
	viper.SetDefault("port", 8000)
	viper.SetDefault("autopump", false)

	viper.BindEnv("port", "PORT")
	viper.BindEnv("autopump", "AUTOPUMP")
	viper.BindEnv("db-url", "DATABASE_URL")
	viper.BindEnv("stellar-core-db-url", "STELLAR_CORE_DATABASE_URL")
	viper.BindEnv("stellar-core-url", "STELLAR_CORE_URL")
	viper.BindEnv("friendbot-secret", "FRIENDBOT_SECRET")
	viper.BindEnv("per-hour-rate-limit", "PER_HOUR_RATE_LIMIT")
	viper.BindEnv("redis-url", "REDIS_URL")
	viper.BindEnv("ruby-horizon-url", "RUBY_HORIZON_URL")
	viper.BindEnv("log-level", "LOG_LEVEL")
	viper.BindEnv("sentry-dsn", "SENTRY_DSN")
	viper.BindEnv("loggly-token", "LOGGLY_TOKEN")
	viper.BindEnv("loggly-host", "LOGGLY_HOST")
	viper.BindEnv("tls-cert", "TLS_CERT")
	viper.BindEnv("tls-key", "TLS_KEY")
	viper.BindEnv("ingest", "INGEST")
	viper.BindEnv("network-passphrase", "NETWORK_PASSPHRASE")
	viper.BindEnv("bank-master-key", "BANK_MASTER_KEY")
	viper.BindEnv("bank-commission-key", "BANK_COMMISSION_KEY")

	viper.BindEnv("restrictions-anonymous-user-max-daily-outcome", "RESTRICTIONS_ANONYMOUS_USER_MAX_DAILY_OUTCOME")
	viper.BindEnv("restrictions-anonymous-user-max-monthly-outcome", "RESTRICTIONS_ANONYMOUS_USER_MAX_MONTHLY_OUTCOME")
	viper.BindEnv("restrictions-anonymous-user-max-annual-outcome", "RESTRICTIONS_ANONYMOUS_USER_MAX_ANNUAL_OUTCOME")
	viper.BindEnv("restrictions-anonymous-user-max-annual-income", "RESTRICTIONS_ANONYMOUS_USER_MAX_ANNUAL_INCOME")
	viper.BindEnv("restrictions-anonymous-user-max-balance", "RESTRICTIONS_ANONYMOUS_USER_MAX_BALANCE")

	rootCmd = &cobra.Command{
		Use:   "horizon",
		Short: "client-facing api server for the stellar network",
		Long:  "client-facing api server for the stellar network",
		Run: func(cmd *cobra.Command, args []string) {
			initApp(cmd, args)
			app.Serve()
		},
	}

	rootCmd.Flags().String(
		"db-url",
		"",
		"horizon postgres database to connect with",
	)

	rootCmd.Flags().String(
		"stellar-core-db-url",
		"",
		"stellar-core postgres database to connect with",
	)

	rootCmd.Flags().String(
		"stellar-core-url",
		"",
		"stellar-core to connect with (for http commands)",
	)

	rootCmd.Flags().Int(
		"port",
		8000,
		"tcp port to listen on for http requests",
	)

	rootCmd.Flags().Bool(
		"autopump",
		false,
		"pump streams every second, instead of once per ledger close",
	)

	rootCmd.Flags().Int(
		"per-hour-rate-limit",
		3600,
		"max count of requests allowed in a one hour period, by remote ip address",
	)

	rootCmd.Flags().String(
		"redis-url",
		"",
		"redis to connect with, for rate limiting",
	)

	rootCmd.Flags().String(
		"log-level",
		"info",
		"Minimum log severity (debug, info, warn, error) to log",
	)

	rootCmd.Flags().String(
		"sentry-dsn",
		"",
		"Sentry URL to which panics and errors should be reported",
	)

	rootCmd.Flags().String(
		"loggly-token",
		"",
		"Loggly token, used to configure log forwarding to loggly",
	)

	rootCmd.Flags().String(
		"loggly-host",
		"",
		"Hostname to be added to every loggly log event",
	)

	rootCmd.Flags().String(
		"friendbot-secret",
		"",
		"Secret seed for friendbot functionality. When empty, friendbot will be disabled",
	)

	rootCmd.Flags().String(
		"tls-cert",
		"",
		"The TLS certificate file to use for securing connections to horizon",
	)

	rootCmd.Flags().String(
		"tls-key",
		"",
		"The TLS private key file to use for securing connections to horizon",
	)

	rootCmd.Flags().Bool(
		"ingest",
		false,
		"causes this horizon process to ingest data from stellar-core into horizon's db",
	)

	rootCmd.Flags().String(
		"network-passphrase",
		"",
		"Override the network passphrase",
	)

	rootCmd.Flags().String(
		"bank-master-key",
		"",
		"Bank's master key",
	)

	rootCmd.Flags().String(
		"bank-commission-key",
		"",
		"Bank's commission key",
	)

	// User restrictions

	rootCmd.Flags().String(
		"restrictions-anonymous-user-max-daily-outcome",
		"500.0",
		"Maximum daily outcome limit for anonymous users.",
	)

	rootCmd.Flags().String(
		"restrictions-anonymous-user-max-monthly-outcome",
		"4000.0",
		"Maximum monthly outcome limit for anonymous users.",
	)

	rootCmd.Flags().String(
		"restrictions-anonymous-user-max-annual-outcome",
		"62000.0",
		"Maximum annual outcome limit for anonymous users.",
	)

	rootCmd.Flags().String(
		"restrictions-anonymous-user-max-annual-income",
		"62000.0",
		"Maximum annual income limit for anonymous users.",
	)

	rootCmd.Flags().String(
		"restrictions-anonymous-user-max-balance",
		"14000.0",
		"Maximum annual income limit for anonymous users.",
	)

	rootCmd.AddCommand(dbCmd)

	viper.BindPFlags(rootCmd.Flags())
}

func initApp(cmd *cobra.Command, args []string) {
	initConfig()

	var err error
	app, err = horizon.NewApp(config)

	if err != nil {
		log.Fatal(err.Error())
	}
}

func initConfig() {
	if viper.GetString("db-url") == "" {
		log.Fatal("Invalid config: db-url is blank.  Please specify --db-url on the command line or set the DATABASE_URL environment variable.")
	}

	if viper.GetString("stellar-core-db-url") == "" {
		log.Fatal("Invalid config: stellar-core-db-url is blank.  Please specify --stellar-core-db-url on the command line or set the STELLAR_CORE_DATABASE_URL environment variable.")
	}

	if viper.GetString("stellar-core-url") == "" {
		log.Fatal("Invalid config: stellar-core-url is blank.  Please specify --stellar-core-url on the command line or set the STELLAR_CORE_URL environment variable.")
	}

	ll, err := logrus.ParseLevel(viper.GetString("log-level"))

	if err != nil {
		log.Fatalf("Could not parse log-level: %v", viper.GetString("log-level"))
	}

	hlog.DefaultLogger.Level = ll

	cert, key := viper.GetString("tls-cert"), viper.GetString("tls-key")

	switch {
	case cert != "" && key == "":
		log.Fatal("Invalid TLS config: key not configured")
	case cert == "" && key != "":
		log.Fatal("Invalid TLS config: cert not configured")
	}

	if viper.GetBool("ingest") && viper.GetString("bank-master-key") == "" {
		log.Fatal("Invalid config: bank-master-key is blank. Please set the BANK_MASTER_KEY environment variable.")
	}
	if viper.GetString("bank-commission-key") == "" {
		log.Fatal("Invalid config: bank-commission-key is blank. Please set the BANK_COMMISSION_KEY environment variable.")
	}

	adminSigValid := viper.GetInt("admin-sig-valid")
	if adminSigValid == 0 {
		adminSigValid = 60
	}

	statisticsTimeout := viper.GetInt("stats-timeout")
	if statisticsTimeout == 0 {
		statisticsTimeout = 60
	}

	processedOpTimeout := viper.GetInt("processed-op-timeout")
	if processedOpTimeout == 0 {
		processedOpTimeout = statisticsTimeout / 2
	}

	config = conf.Config{
		DatabaseURL:               viper.GetString("db-url"),
		StellarCoreDatabaseURL:    viper.GetString("stellar-core-db-url"),
		StellarCoreURL:            viper.GetString("stellar-core-url"),
		Autopump:                  viper.GetBool("autopump"),
		Port:                      viper.GetInt("port"),
		RateLimit:                 getRateLimit(),
		RedisURL:                  viper.GetString("redis-url"),
		LogLevel:                  ll,
		SentryDSN:                 viper.GetString("sentry-dsn"),
		LogglyToken:               viper.GetString("loggly-token"),
		LogglyHost:                viper.GetString("loggly-host"),
		FriendbotSecret:           viper.GetString("friendbot-secret"),
		TLSCert:                   cert,
		TLSKey:                    key,
		Ingest:                    viper.GetBool("ingest"),
		BankMasterKey:             viper.GetString("bank-master-key"),
		BankCommissionKey:         viper.GetString("bank-commission-key"),
		AnonymousUserRestrictions: getAnonymousUserRestrictions(),
		AdminSignatureValid:       time.Duration(adminSigValid) * time.Second,
		StatisticsTimeout:         time.Duration(statisticsTimeout) * time.Second,
		ProcessedOpTimeout:        time.Duration(processedOpTimeout) * time.Second,
	}
}

func getRateLimit() *throttled.RateQuota {
	limitPerHour := viper.GetInt("per-hour-rate-limit")
	if limitPerHour <= 0 {
		return nil
	}
	return &throttled.RateQuota{
		MaxRate:  throttled.PerHour(limitPerHour),
		MaxBurst: 1,
	}
}

func getAnonymousUserRestrictions() conf.AnonymousUserRestrictions {
	var restrictions conf.AnonymousUserRestrictions
	var value int64
	var err error

	value, err = parseAmount(viper.GetString("restrictions-anonymous-user-max-daily-outcome"))
	if err != nil {
		log.Fatalf(
			"Could not parse restrictions-anonymous-user-max-daily-outcome: %v",
			viper.GetString("restrictions-anonymous-user-max-daily-outcome"),
		)
	}
	restrictions.MaxDailyOutcome = value

	value, err = parseAmount(viper.GetString("restrictions-anonymous-user-max-monthly-outcome"))
	if err != nil {
		log.Fatalf(
			"Could not parse restrictions-anonymous-user-max-monthly-outcome: %v",
			viper.GetString("restrictions-anonymous-user-max-monthly-outcome"),
		)
	}
	restrictions.MaxMonthlyOutcome = value

	value, err = parseAmount(viper.GetString("restrictions-anonymous-user-max-annual-outcome"))
	if err != nil {
		log.Fatalf(
			"Could not parse restrictions-anonymous-user-max-annual-outcome: %v",
			viper.GetString("restrictions-anonymous-user-max-annual-outcome"),
		)
	}
	restrictions.MaxAnnualOutcome = value

	value, err = parseAmount(viper.GetString("restrictions-anonymous-user-max-balance"))
	if err != nil {
		log.Fatalf(
			"Could not parse restrictions-anonymous-user-max-balance: %v",
			viper.GetString("restrictions-anonymous-user-max-balance"),
		)
	}
	restrictions.MaxBalance = value

	return restrictions
}

func parseAmount(strAmount string) (int64, error) {
	xdrAmount, err := amount.Parse(strAmount)
	intAmount := int64(xdrAmount)

	return intAmount, err
}
