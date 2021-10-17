package models

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/go-redis/redis"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// RelayConfig contains valid configuration.
type RelayConfig struct {
	actorKey        *rsa.PrivateKey
	domain          *url.URL
	redisClient     *redis.Client
	brokerURL       *url.URL
	redisURL        *url.URL
	serverBind      string
	serviceName     string
	serviceSummary  string
	serviceIconURL  *url.URL
	serviceImageURL *url.URL
	jobConcurrency  int
}

// NewRelayConfig create valid RelayConfig from viper configuration. If invalid configuration detected, return error.
func NewRelayConfig() (*RelayConfig, error) {
	domain, err := url.ParseRequestURI("https://" + viper.GetString("RELAY_DOMAIN"))
	if err != nil {
		return nil, errors.New("RELAY_DOMAIN: " + err.Error())
	}

	iconURL, err := url.ParseRequestURI(viper.GetString("RELAY_ICON"))
	if err != nil {
		logrus.Warn("RELAY_ICON: INVALID OR EMPTY. THIS COLUMN IS DISABLED.")
		iconURL = nil
	}

	imageURL, err := url.ParseRequestURI(viper.GetString("RELAY_IMAGE"))
	if err != nil {
		logrus.Warn("RELAY_IMAGE: INVALID OR EMPTY. THIS COLUMN IS DISABLED.")
		imageURL = nil
	}

	jobConcurrency := viper.GetInt("JOB_CONCURRENCY")
	if jobConcurrency < 1 {
		return nil, errors.New("JOB_CONCURRENCY IS 0 OR EMPTY. SHOULD BE MORE THAN 1")
	}

	privateKey, err := readPrivateKeyRSA(viper.GetString("ACTOR_PEM"))
	if err != nil {
		return nil, errors.New("ACTOR_PEM: " + err.Error())
	}

	redisURL, err := url.ParseRequestURI(viper.GetString("REDIS_URL"))
	if err != nil {
		return nil, errors.New("REDIS_URL: " + err.Error())
	}
	redisOption, err := redis.ParseURL(redisURL.String())
	if err != nil {
		return nil, errors.New("REDIS_URL: " + err.Error())
	}
	redisClient := redis.NewClient(redisOption)
	err = redisClient.Ping().Err()
	if err != nil {
		return nil, errors.New("Redis Connection Test: " + err.Error())
	}

	brokerURL, err := url.ParseRequestURI(viper.GetString("BROKER_URL"))
	if err != nil {
		logrus.Warn("BROKER_URL: INVALID OR EMPTY. USE REDIS_URL.")
		brokerURL = redisURL
	}

	serverBind := viper.GetString("RELAY_BIND")

	return &RelayConfig{
		actorKey:        privateKey,
		domain:          domain,
		redisClient:     redisClient,
		brokerURL:       brokerURL,
		redisURL:        redisURL,
		serverBind:      serverBind,
		serviceName:     viper.GetString("RELAY_SERVICENAME"),
		serviceSummary:  viper.GetString("RELAY_SUMMARY"),
		serviceIconURL:  iconURL,
		serviceImageURL: imageURL,
		jobConcurrency:  jobConcurrency,
	}, nil
}

// ServerBind is API Server's bind interface definition.
func (relayConfig *RelayConfig) ServerBind() string {
	return relayConfig.serverBind
}

// ServerHostname is API Server's hostname definition.
func (relayConfig *RelayConfig) ServerHostname() *url.URL {
	return relayConfig.domain
}

// ServerServiceName is API Server's servername definition.
func (relayConfig *RelayConfig) ServerServiceName() string {
	return relayConfig.serviceName
}

// JobConcurrency is API Worker's jobConcurrency definition.
func (relayConfig *RelayConfig) JobConcurrency() int {
	return relayConfig.jobConcurrency
}

// ActorKey is API Worker's HTTPSignature private key.
func (relayConfig *RelayConfig) ActorKey() *rsa.PrivateKey {
	return relayConfig.actorKey
}

// RedisClient is return redis client from RelayConfig.
func (relayConfig *RelayConfig) RedisClient() *redis.Client {
	return relayConfig.redisClient
}

// DumpWelcomeMessage provide build and config information string.
func (relayConfig *RelayConfig) DumpWelcomeMessage(moduleName string, version string) string {
	return fmt.Sprintf(`Welcome to YUKIMOCHI Activity-Relay %s - %s
 - Configuration
RELAY NAME      : %s
RELAY DOMAIN    : %s
BROKER URL      : %s
REDIS URL       : %s
BIND ADDRESS    : %s
JOB_CONCURRENCY : %s
`, version, moduleName, relayConfig.serviceName, relayConfig.domain.Host, relayConfig.brokerURL, relayConfig.redisURL, relayConfig.serverBind, strconv.Itoa(relayConfig.jobConcurrency))
}

// NewMachineryServer create Redis backed Machinery Server from RelayConfig.
func NewMachineryServer(globalConfig *RelayConfig) (*machinery.Server, error) {
	cnf := &config.Config{
		Broker:          globalConfig.brokerURL.String(),
		DefaultQueue:    "relay",
		ResultBackend:   globalConfig.redisURL.String(),
		ResultsExpireIn: 1,
		AMQP: &config.AMQPConfig{
			Exchange:     "relay_exchange",
			ExchangeType: "direct",
			BindingKey:   "relay_task",
		},
	}
	newServer, err := machinery.NewServer(cnf)

	return newServer, err
}
