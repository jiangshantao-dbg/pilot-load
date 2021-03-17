package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/howardjohn/pilot-load/pkg/kube"
	"github.com/howardjohn/pilot-load/pkg/simulation/model"
	"github.com/howardjohn/pilot-load/pkg/simulation/security"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"istio.io/pkg/log"
)

var (
	pilotAddress   = defaultAddress()
	xdsMetadata    = map[string]string{}
	auth           = string(security.AuthTypeDefault)
	kubeconfig     = os.Getenv("KUBECONFIG")
	loggingOptions = defaultLogOptions()

	authTrustDomain   = ""
	authClusterUrl    = ""
	authProjectNumber = ""

	qps = 100
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&pilotAddress, "pilot-address", "p", pilotAddress, "address to pilot")
	rootCmd.PersistentFlags().StringVarP(&auth, "auth", "a", auth,
		fmt.Sprintf("auth type use. If not set, default based on port number. Supported options: %v", security.AuthTypeOptions()))
	rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", kubeconfig, "kubeconfig")
	rootCmd.PersistentFlags().IntVar(&qps, "qps", qps, "qps for kube client")
	rootCmd.PersistentFlags().StringToStringVarP(&xdsMetadata, "metadata", "m", xdsMetadata, "xds metadata")

	rootCmd.PersistentFlags().StringVar(&authClusterUrl, "clusterURL", authClusterUrl, "cluster URL (for google auth)")
	rootCmd.PersistentFlags().StringVar(&authTrustDomain, "trustDomain", authTrustDomain, "trust domain (for google auth)")
	rootCmd.PersistentFlags().StringVar(&authProjectNumber, "projectNumber", authProjectNumber, "project number (for google auth)")
}

func defaultAddress() string {
	_, inCluster := os.LookupEnv("KUBERNETES_SERVICE_HOST")
	if inCluster {
		return "istiod.istio-system.svc:15010"
	}
	return "localhost:15010"
}

func defaultLogOptions() *log.Options {
	o := log.DefaultOptions()

	// These scopes are, at the default "INFO" level, too chatty for command line use
	o.SetOutputLevel("dump", log.WarnLevel)
	o.SetOutputLevel("token", log.ErrorLevel)

	return o
}

func GetArgs() (model.Args, error) {
	if qps == 0 {
		qps = 100
	}
	if kubeconfig == "" {
		kubeconfig = filepath.Join(os.Getenv("HOME"), "/.kube/config")
	}
	cl, err := kube.NewClient(kubeconfig, qps)
	if err != nil {
		return model.Args{}, err
	}
	auth := security.AuthType(auth)
	if auth == "" {
		auth = security.DefaultAuthForAddress(pilotAddress)
	}
	authOpts := &security.AuthOptions{
		Type:          auth,
		Client:        cl,
		TrustDomain:   authTrustDomain,
		ProjectNumber: authProjectNumber,
		ClusterURL:    authClusterUrl,
	}
	if err := authOpts.AutoPopulate(); err != nil {
		return model.Args{}, err
	}
	return model.Args{
		PilotAddress: pilotAddress,
		Metadata:     xdsMetadata,
		Client:       cl,
		Auth:         authOpts,
	}, nil
}

var rootCmd = &cobra.Command{
	Use:          "pilot-load",
	Short:        "open XDS connections to pilot",
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return log.Configure(loggingOptions)
	},
}

func logConfig(config interface{}) {
	bytes, err := yaml.Marshal(config)
	if err != nil {
		panic(err.Error())
	}
	log.Infof("Starting simulation with config:\n%v", string(bytes))
}

func init() {
	rootCmd.AddCommand(
		adscCmd,
		clusterCmd,
		impersonateCmd,
		proberCmd,
		startupCmd,
	)
}

func Execute() {
	loggingOptions.AttachCobraFlags(rootCmd)
	hiddenFlags := []string{
		"log_as_json", "log_rotate", "log_rotate_max_age", "log_rotate_max_backups",
		"log_rotate_max_size", "log_stacktrace_level", "log_target", "log_caller",
	}
	for _, opt := range hiddenFlags {
		_ = rootCmd.PersistentFlags().MarkHidden(opt)
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
