package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/CiscoCloud/mantl-api/api"
	"github.com/CiscoCloud/mantl-api/install"
	"github.com/CiscoCloud/mantl-api/marathon"
	"github.com/CiscoCloud/mantl-api/mesos"
	"github.com/CiscoCloud/mantl-api/utils/http"
	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const Name = "mantl-api"
const Version = "0.2.0"

var wg sync.WaitGroup

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
	rootCmd.PersistentFlags().Bool("consul-no-verify-ssl", false, "Consul SSL verification")
	rootCmd.PersistentFlags().String("marathon", "", "Marathon Api address")
	rootCmd.PersistentFlags().String("marathon-user", "", "Marathon Api user")
	rootCmd.PersistentFlags().String("marathon-password", "", "Marathon Api password")
	rootCmd.PersistentFlags().Bool("marathon-no-verify-ssl", false, "Marathon SSL verification")
	rootCmd.PersistentFlags().String("mesos", "", "Mesos Api address")
	rootCmd.PersistentFlags().String("mesos-principal", "", "Mesos principal for framework authentication")
	rootCmd.PersistentFlags().String("mesos-secret", "", "Deprecated. Use mesos-secret-path instead")
	rootCmd.PersistentFlags().String("mesos-secret-path", "/etc/sysconfig/mantl-api", "Path to a file on host sytem that contains the mesos secret for framework authentication")
	rootCmd.PersistentFlags().Bool("mesos-no-verify-ssl", false, "Mesos SSL verification")
	rootCmd.PersistentFlags().String("listen", ":4001", "mantl-api listen address")
	rootCmd.PersistentFlags().String("zookeeper", "", "Comma-delimited list of zookeeper servers")
	rootCmd.PersistentFlags().Bool("force-sync", false, "Force a synchronization of all sources")
	rootCmd.PersistentFlags().String("config-file", "", "The path to a configuration file")
	rootCmd.PersistentFlags().Int("consul-refresh-interval", 10, "The number of seconds after which to check consul for package requests")

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
			syncRepo(nil, true)
		},
	}
	rootCmd.AddCommand(syncCommand)

	versionCommand := &cobra.Command{
		Use:   "version",
		Short: fmt.Sprintf("Print the version number of %s", Name),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s v%s\n", Name, Version)
		},
	}
	rootCmd.AddCommand(versionCommand)

	rootCmd.Execute()
}

func start() {
	log.Infof("Starting %s v%s", Name, Version)
	client := consulClient()

	marathonUrl := viper.GetString("marathon")
	if marathonUrl == "" {
		marathonHosts := NewDiscovery(client, "marathon", "").discoveredHosts
		if len(marathonHosts) > 0 {
			marathonUrl = fmt.Sprintf("http://%s", marathonHosts[0])
		} else {
			marathonUrl = "http://localhost:8080"
		}
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
		mesosHosts := NewDiscovery(client, "mesos", "leader").discoveredHosts
		if len(mesosHosts) > 0 {
			mesosUrl = fmt.Sprintf("http://%s", mesosHosts[0])
		} else {
			mesosUrl = "http://locahost:5050"
		}
	}
	mesosClient, err := mesos.NewMesos(
		mesosUrl,
		viper.GetString("mesos-principal"),
		viper.GetString("mesos-secret-path"),
		viper.GetBool("mesos-no-verify-ssl"),
	)
	if err != nil {
		log.Fatalf("Could not create mesos client: %v", err)
	}

	var zkHosts []string
	zkUrls := viper.GetString("zookeeper")
	if zkUrls == "" {
		zkHosts = NewDiscovery(client, "zookeeper", "").discoveredHosts
		if len(zkHosts) == 0 {
			zkHosts = []string{"locahost:2181"}
		}
	} else {
		zkHosts = strings.Split(zkUrls, ",")
	}

	inst, err := install.NewInstall(client, marathonClient, mesosClient, zkHosts)
	if err != nil {
		log.Fatalf("Could not create install client: %v", err)
	}

	// sync sources to consul
	syncRepo(inst, viper.GetBool("force-sync"))

	wg.Add(1)
	go inst.Watch(time.Duration(viper.GetInt("consul-refresh-interval")))
	go api.NewApi(Name, viper.GetString("listen"), inst, mesosClient, wg).Start()
	wg.Wait()
}

func consulClient() *consul.Client {
	consulConfig := consul.DefaultConfig()
	scheme, address, _, err := http.ParseUrl(viper.GetString("consul"))
	if err != nil {
		log.Fatalf("Could not create consul client: %v", err)
	}
	consulConfig.Scheme = scheme
	consulConfig.Address = address

	log.Debugf("Using Consul at %s over %s", consulConfig.Address, consulConfig.Scheme)

	if viper.GetBool("consul-no-verify-ssl") {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		consulConfig.HttpClient.Transport = transport
	}

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

func syncRepo(inst *install.Install, force bool) {
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
			Branch:     "version-0.7",
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

	if err := inst.SyncSources(sources, force); err != nil {
		log.Fatal(err)
	}
}

func readConfigFile() {
	// read configuration file if specified
	configFile := viper.GetString("config-file")
	if configFile != "" {
		configFile = os.ExpandEnv(configFile)
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
