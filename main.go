package main

import (
	"github.com/CiscoCloud/mantl-api/api"
	"github.com/CiscoCloud/mantl-api/install"
	"github.com/CiscoCloud/mantl-api/marathon"
	"github.com/CiscoCloud/mantl-api/mesos"
	"github.com/CiscoCloud/mantl-api/zookeeper"
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"net/url"
	"strings"
)

const Name = "mantl-api"
const Version = "0.1.0"

func start() {
	consulConfig := consul.DefaultConfig()
	scheme, address := parseConsulAddress(viper.GetString("consul"))
	consulConfig.Scheme = scheme
	consulConfig.Address = address

	log.Debugf("Using Consul at %s over %s", consulConfig.Address, consulConfig.Scheme)

	client, err := consul.NewClient(consulConfig)
	if err != nil {
		log.Fatalf("Could not create consul client: %v", err)
	}

	// abort if we cannot connect to consul
	err = testConsul(client)
	if err != nil {
		log.Fatalf("Could not connect to consul: %v", err)
	}

	scheme, address = parseMarathonAddress(viper.GetString("marathon"))
	marathonClient := marathon.NewMarathon(
		address,
		scheme,
		viper.GetString("marathon-user"),
		viper.GetString("marathon-password"),
		viper.GetBool("marathon-no-verify-ssl"),
	)
	if err != nil {
		log.Fatalf("Could not get marathon client: %v", err)
	}

	scheme, address = parseMesosAddress(viper.GetString("mesos"))
	mesosClient := mesos.NewMesos(
		address,
		scheme,
		viper.GetString("mesos-principal"),
		viper.GetString("mesos-secret"),
		viper.GetBool("mesos-no-verify-ssl"),
	)
	if err != nil {
		log.Fatalf("Could not get mesos client: %v", err)
	}

	zkServers := strings.Split(viper.GetString("zookeeper"), ",")
	zk := zookeeper.NewZookeeper(zkServers)

	inst := install.NewInstall(client, marathonClient, mesosClient, zk)

	// sync sources to consul
	sources := []*install.Source{
		&install.Source{
			Name:       "mantl",
			Path:       "https://github.com/CiscoCloud/mantl-universe.git",
			SourceType: install.Git,
			Index:      1,
		},
		&install.Source{
			Name:       "mesosphere",
			Path:       "https://github.com/mesosphere/universe.git",
			SourceType: install.Git,
			Index:      0,
		},
	}
	inst.SyncSources(sources, viper.GetBool("force-sync"))

	// start listener
	api.NewApi(viper.GetString("listen"), inst).Start()
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "mantl-api",
		Short: "runs the mantl-api",
		Run: func(cmd *cobra.Command, args []string) {
			start()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			setupLogging()
		},
	}

	rootCmd.PersistentFlags().String("log-level", "info", "one of debug, info, warn, error, or fatal")
	rootCmd.PersistentFlags().String("log-format", "text", "specify output (text or json)")
	rootCmd.PersistentFlags().String("consul", "http://localhost:8500", "Consul Api address")
	rootCmd.PersistentFlags().String("marathon", "http://localhost:8080", "Marathon Api address")
	rootCmd.PersistentFlags().String("marathon-user", "", "Marathon Api user")
	rootCmd.PersistentFlags().String("marathon-password", "", "Marathon Api password")
	rootCmd.PersistentFlags().Bool("marathon-no-verify-ssl", false, "Marathon SSL verification")
	rootCmd.PersistentFlags().String("mesos", "http://localhost:5050", "Mesos Api address")
	rootCmd.PersistentFlags().String("mesos-principal", "", "Mesos principal for framework authentication")
	rootCmd.PersistentFlags().String("mesos-secret", "", "Mesos secret for framework authentication")
	rootCmd.PersistentFlags().Bool("mesos-no-verify-ssl", false, "Mesos SSL verification")
	rootCmd.PersistentFlags().String("listen", ":4001", "mantl-api listen address")
	rootCmd.PersistentFlags().String("zookeeper", "localhost:2181", "Comma-delimited list of zookeeper servers")
	rootCmd.PersistentFlags().Bool("force-sync", false, "Force a synchronization of all sources")

	for _, flags := range []*pflag.FlagSet{rootCmd.PersistentFlags()} {
		err := viper.BindPFlags(flags)
		if err != nil {
			log.WithField("error", err).Fatal("could not bind flags")
		}
	}

	viper.SetEnvPrefix("mantl_api")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	rootCmd.Execute()
}

func testConsul(client *consul.Client) error {
	kv := client.KV()
	_, _, err := kv.Get("mantl-install", nil)
	return err
}

func parseConsulAddress(u string) (scheme string, host string) {
	return parseAddress(u, "http", "localhost:8500")
}

func parseMarathonAddress(u string) (scheme string, host string) {
	return parseAddress(u, "http", "localhost:8080")
}

func parseMesosAddress(u string) (scheme string, host string) {
	return parseAddress(u, "http", "localhost:5050")
}

func parseAddress(u string, defaultScheme string, defaultHost string) (scheme string, host string) {
	url, err := url.Parse(u)
	if err != nil {
		log.Fatalf("Could not parse address %s: %v", u, err)
		return "", ""
	}

	scheme = url.Scheme
	host = url.Host

	if scheme == "" {
		log.Warnf("Could not parse scheme. Using '%s'", defaultScheme)
		scheme = defaultScheme
	}

	if host == "" {
		log.Warnf("Could not parse host. Using '%s'", defaultHost)
		host = defaultHost
	}

	return scheme, host
}

func setupLogging() {
	switch viper.GetString("log-level") {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	default:
		log.WithField("log-level", viper.GetString("log-level")).Warning("invalid log level. defaulting to info.")
		log.SetLevel(log.InfoLevel)
	}

	switch viper.GetString("log-format") {
	case "text":
		log.SetFormatter(new(log.TextFormatter))
	case "json":
		log.SetFormatter(new(log.JSONFormatter))
	default:
		log.WithField("log-format", viper.GetString("log-format")).Warning("invalid log format. defaulting to text.")
		log.SetFormatter(new(log.TextFormatter))
	}
}
