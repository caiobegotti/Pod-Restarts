package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/caiobegotti/pod-restarts/pkg/plugin"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	KubernetesConfigFlags *genericclioptions.ConfigFlags
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pod-restarts",
		Short: "Sorted table of all pods with restarts and their age, start time.",
		Long: `This command prints a table with all the restarting pods inside
your cluster and the lookup can be restricted to a specific namespace, based
on a minimum threshold for restarts or just count containers restarts too.

The purpose of this is to have a glance at what has been failing and since
when, as age and start times are included in the result table. The alternative to
that would be to run multiple shell commands with complex parsing or plot N graphs
with Prometheus or other tool.`,
		Example: `
Cluster-wide listing
$ kubectl pod-restarts

Restricts listing to a namespace (faster in big clusters)
$ kubectl pod-restarts -n production

Ignores pods below a specific threshold (10 restarts)
$ kubectl pod-restarts -t 10

Also lists all the containers restarting inside the pods
$ kubectl pod-restarts -c`,
		SilenceErrors: true,
		SilenceUsage:  false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := plugin.RunPlugin(KubernetesConfigFlags); err != nil {
				return errors.Cause(err)
			}

			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	KubernetesConfigFlags = genericclioptions.NewConfigFlags(false)
	KubernetesConfigFlags.AddFlags(cmd.Flags())

	// extra flags to our plugin
	cmd.Flags().BoolP("containers", "c", false, "Also lists containers restarts, ignoring thresholds")
	cmd.Flags().Int32P("threshold", "t", 0, "Only list restarts above the given threshold")

	// hide common flags supported by any kubectl command to declutter -h/--help
	// most people would only (if ever) miss kubeconfig, context or cluster
	cmd.Flags().MarkHidden("as-group")
	cmd.Flags().MarkHidden("as")
	cmd.Flags().MarkHidden("cache-dir")
	cmd.Flags().MarkHidden("certificate-authority")
	cmd.Flags().MarkHidden("client-certificate")
	cmd.Flags().MarkHidden("client-key")
	cmd.Flags().MarkHidden("cluster")
	cmd.Flags().MarkHidden("context")
	cmd.Flags().MarkHidden("insecure-skip-tls-verify")
	cmd.Flags().MarkHidden("kubeconfig")
	cmd.Flags().MarkHidden("password")
	cmd.Flags().MarkHidden("request-timeout")
	cmd.Flags().MarkHidden("server")
	cmd.Flags().MarkHidden("token")
	cmd.Flags().MarkHidden("user")
	cmd.Flags().MarkHidden("username")

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	return cmd
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.AutomaticEnv()
}
