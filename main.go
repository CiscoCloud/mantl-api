package main

import (
	"github.com/CiscoCloud/mantl-api/api"
	"github.com/CiscoCloud/mantl-api/install"
	"github.com/CiscoCloud/mantl-api/marathon"
	"github.com/CiscoCloud/mantl-api/mesos"
	"github.com/CiscoCloud/mantl-api/utils/http"
	"github.com/CiscoCloud/mantl-api/zookeeper"
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"strings"
)

const Name = "mantl-api"
const Version = "0.1.3.1"

func main() {
	rootCmd := &cobra.Command{
		Use:   "mantl-api",
		Short: "runs the mantl-api",
		Run: func(cmd *cobra.Command, args []string) {
			start()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			readConfigFile()
			setupLogging()
		},
	}

	rootCmd.PersistentFlags().String("log-level", "info", "one of debug, info, warn, error, or fatal")
	rootCmd.PersistentFlags().String("log-format", "text", "specify output (text or json)")
	rootCmd.PersistentFlags().String("consul", "http://localhost:8500", "Consul Api address")
	rootCmd.PersistentFlags().String("marathon", "", "Marathon Api address")
	rootCmd.PersistentFlags().String("marathon-user", "", "Marathon Api user")
	rootCmd.PersistentFlags().String("marathon-password", "", "Marathon Api password")
	rootCmd.PersistentFlags().Bool("marathon-no-verify-ssl", false, "Marathon SSL verification")
	rootCmd.PersistentFlags().String("mesos", "", "Mesos Api address")
	rootCmd.PersistentFlags().String("mesos-principal", "", "Mesos principal for framework authentication")
	rootCmd.PersistentFlags().String("mesos-secret", "", "Mesos secret for framework authentication")
	rootCmd.PersistentFlags().Bool("mesos-no-verify-ssl", false, "Mesos SSL verification")
	rootCmd.PersistentFlags().String("listen", ":4001", "mantl-api listen address")
	rootCmd.PersistentFlags().String("zookeeper", "", "Comma-delimited list of zookeeper servers")
	rootCmd.PersistentFlags().Bool("force-sync", false, "Force a synchronization of all sources")
	rootCmd.PersistentFlags().String("config-file", "", "The path to a configuration file")

	for _, flags := range []*pflag.FlagSet{rootCmd.PersistentFlags()} {
		err := viper.BindPFlags(flags)
		if err != nil {
			log.WithField("error", err).Fatal("could not bind flags")
		}
	}

	viper.SetEnvPrefix("mantl_api")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	syncCommand := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize universe repositories",
		Long:  "Forces a synchronization of all configured sources",
		Run: func(cmd *cobra.Command, args []string) {
			sync(nil, true)
		},
	}
	rootCmd.AddCommand(syncCommand)

	rootCmd.Execute()
}

func start() {
	client := consulClient()

	marathonUrl := viper.GetString("marathon")
	if marathonUrl == "" {
		marathonUrl = NewDiscovery(client, "marathon", "", "http", marathonUrl).discoveredUrl
	}
	marathonClient, err := marathon.NewMarathon(
		marathonUrl,
		viper.GetString("marathon-user"),
		viper.GetString("marathon-password"),
		viper.GetBool("marathon-no-verify-ssl"),
	)
	if err != nil {
		log.Fatalf("Could not create marathon client: %v", err)
	}

	mesosUrl := viper.GetString("mesos")
	if mesosUrl == "" {
		mesosUrl = NewDiscovery(client, "mesos", "leader", "http", "http://localhost:5050").discoveredUrl
	}
	mesosClient, err := mesos.NewMesos(
		mesosUrl,
		viper.GetString("mesos-principal"),
		viper.GetString("mesos-secret"),
		viper.GetBool("mesos-no-verify-ssl"),
	)
	if err != nil {
		log.Fatalf("Could not create mesos client: %v", err)
	}

	zkUrls := viper.GetString("zookeeper")
	if zkUrls == "" {
		zkUrls = NewDiscovery(client, "zookeeper", "", "", "localhost:2181").discoveredUrl
	}
	zkServers := strings.Split(zkUrls, ",")
	zk := zookeeper.NewZookeeper(zkServers)

	inst, err := install.NewInstall(client, marathonClient, mesosClient, zk)
	if err != nil {
		log.Fatalf("Could not create install client: %v", err)
	}

	// sync sources to consul
	sync(inst, viper.GetBool("force-sync"))

	// start listener
	api.NewApi(viper.GetString("listen"), inst).Start()
}

func consulClient() *consul.Client {
	consulConfig := consul.DefaultConfig()
	scheme, address, err := http.ParseUrl(viper.GetString("consul"))
	if err != nil {
		log.Fatalf("Could not create consul client: %v", err)
	}
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

	return client
}

func testConsul(client *consul.Client) error {
	kv := client.KV()
	_, _, err := kv.Get("mantl-install", nil)
	return err
}

func sync(inst *install.Install, force bool) {
	var err error
	if inst == nil {
		client := consulClient()
		inst, err = install.NewInstall(client, nil, nil, nil)
		if err != nil {
			log.Fatalf("Could not create install client: %v", err)
		}
	}

	defaultSources := []*install.Source{
		&install.Source{
			Name:       "mantl",
			Path:       "https://github.com/CiscoCloud/mantl-universe.git",
			SourceType: install.Git,
			Branch:     "master",
			Index:      1,
		},
		&install.Source{
			Name:       "mesosphere",
			Path:       "https://github.com/mesosphere/universe.git",
			SourceType: install.Git,
			Index:      0,
		},
	}

	sources := []*install.Source{}

	configuredSources := viper.GetStringMap("sources")

	if len(configuredSources) > 0 {
		for name, val := range configuredSources {
			source := &install.Source{Name: name, SourceType: install.FileSystem}
			sourceConfig := val.(map[string]interface{})

			if path, ok := sourceConfig["path"].(string); ok {
				source.Path = path
			}

			if index, ok := sourceConfig["index"].(int64); ok {
				source.Index = int(index)
			}

			if sourceType, ok := sourceConfig["type"].(string); ok {
				if strings.EqualFold(sourceType, "git") {
					source.SourceType = install.Git
				}
			}

			if branch, ok := sourceConfig["branch"].(string); ok {
				source.Branch = branch
			}

			if source.IsValid() {
				sources = append(sources, source)
			} else {
				log.Warnf("Invalid source configuration for %s", name)
			}
		}
	}

	if len(sources) == 0 {
		sources = defaultSources
	}

	inst.SyncSources(sources, force)
}

func readConfigFile() {
	// read configuration file if specified
	configFile := viper.GetString("config-file")
	if configFile != "" {
		if _, err := os.Stat(configFile); err == nil {
			viper.SetConfigFile(configFile)
			err = viper.ReadInConfig()
			if err != nil {
				log.Warnf("Could not read configuration file: %v", err)
			}
		} else {
			log.Warnf("Could not find configuration file: %s", configFile)
		}
	}
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
