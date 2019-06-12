package main

import (
	golog "log"
	"os"

	"code.cloudfoundry.org/cf-operator/version"
	eirinix "github.com/SUSE/eirinix"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	helloworld "github.com/SUSE/eirinix-sample/helloworld"
)

var log *zap.SugaredLogger
var rootCmd = &cobra.Command{
	Use:   "eirinix-sample",
	Short: "hello-world extension for Eirini",
	Run: func(cmd *cobra.Command, args []string) {
		defer log.Sync()
		kubeConfig := viper.GetString("kubeconfig")

		namespace := viper.GetString("namespace")

		log.Infof("Starting %s with namespace %s", version.Version, namespace)

		webhookHost := viper.GetString("operator-webhook-host")
		webhookPort := viper.GetInt32("operator-webhook-port")

		if webhookHost == "" {
			log.Fatal("required flag 'operator-webhook-host' not set (env variable: OPERATOR_WEBHOOK_HOST)")
		}
		x := eirinix.NewManager(
			eirinix.ManagerOptions{
				Namespace:           namespace,
				Host:                webhookHost,
				Port:                webhookPort,
				KubeConfig:          kubeConfig,
				Logger:              log,
				OperatorFingerprint: "eirini-x-helloworld",
			})

		x.AddExtension(helloworld.New())
		log.Fatal(x.Start())
	},
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		golog.Fatalf("cannot initialize ZAP logger: %v", err)
	}
	log = logger.Sugar()

	pf := rootCmd.PersistentFlags()

	pf.StringP("kubeconfig", "c", "", "Path to a kubeconfig, not required in-cluster")
	pf.StringP("namespace", "n", "eirini", "Namespace to watch for Eirini apps")
	pf.StringP("operator-webhook-host", "w", "", "Hostname/IP under which the webhook server can be reached from the cluster")
	pf.StringP("operator-webhook-port", "p", "2999", "Port the webhook server listens on")
	viper.BindPFlag("kubeconfig", pf.Lookup("kubeconfig"))
	viper.BindPFlag("namespace", pf.Lookup("namespace"))
	viper.BindPFlag("operator-webhook-host", pf.Lookup("operator-webhook-host"))
	viper.BindPFlag("operator-webhook-port", pf.Lookup("operator-webhook-port"))
	viper.BindEnv("kubeconfig")
	viper.BindEnv("namespace", "NAMESPACE")
	viper.BindEnv("operator-webhook-host", "OPERATOR_WEBHOOK_HOST")
	viper.BindEnv("operator-webhook-port", "OPERATOR_WEBHOOK_PORT")

	if err := rootCmd.Execute(); err != nil {
		golog.Fatal(err)
		os.Exit(1)
	}
}
