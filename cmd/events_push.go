package cmd

import (
	"fmt"

	"github.com/Patrick-Ivann/observability-pusher/internal/kubernetes"

	"github.com/spf13/cobra"
)

func init() {
	eventsPushCmd.Flags().String("name", "", "Name of producing app")
	eventsPushCmd.Flags().String("namespace", "default", "Namespace to create the app in")
	eventsPushCmd.Flags().String("message", "", "Message to print at regular intervals. IF a value is provided to event-id, this flag will fill the message template e.g '--message=value1,value2'")
	eventsPushCmd.Flags().Int("interval", 5, "interval")
	eventsPushCmd.Flags().Var(&podLabels, "pod-labels", `Specify labels as "key:value,anotherkey:anothervalue"`)
	eventsPushCmd.Flags().Bool("psa-enabled", false, "if the cluster has some Pod Security Admission enabled")

	eventsPushCmd.Flags().String("image-pull-secret", "", "Name of the image pull secret")
	eventsPushCmd.Flags().String("registry-path", "", "Registry path for the image")
	eventsPushCmd.Flags().String("service-account", "default", "Name of the ServiceAccount to use")

}

// eventsPushCmd represents the push command for events
var eventsPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push an event",
	Long:  "Push an event --element <> --event-id=<> --message=something --namespace=<> --interval=10 --pod-labels=bip:boup",
	Run: func(cmd *cobra.Command, args []string) {

		namespace, _ := cmd.Flags().GetString("namespace")
		applicationName, _ := cmd.Flags().GetString("name")
		message, _ := cmd.Flags().GetString("message")
		intervalInSecond, _ := cmd.Flags().GetInt("interval")
		isPsaEnabled, _ := cmd.Flags().GetBool("psa-enabled")
		registry, _ := cmd.Flags().GetString("registry-path")
		registryPullSecret, _ := cmd.Flags().GetString("image-pull-secret")
		serviceAccount, _ := cmd.Flags().GetString("service-account")

		podLabels.Append(Labels{"obs-pusher": "events"})

		knImpl, err := kubernetes.NewClientset(registry, registryPullSecret, serviceAccount)
		if err != nil {
			println(err.Error())
			return
		}

		// check if namespace exists
		isNamespaceExisting, err := knImpl.IsNamespaceExisting(namespace)
		if err != nil {
			println(err.Error())
			return
		}
		// create namespace
		if !isNamespaceExisting {
			knImpl.CreateNamespace(namespace)
		}

		// Check if pod exists by fetching it based on labels
		podList, err := knImpl.FetchPodByLabels(namespace, Labels{"obs-pusher": "events"})
		if err != nil {
			println(err.Error())
			return
		}

		// delete existing pod if it exists
		if len(podList.Items) > 0 {
			for _, pod := range podList.Items {
				knImpl.DeletePod(namespace, pod.Name)
				knImpl.WaitForPodDeletion(namespace, pod.Name)
			}
		}
		// Create a new pod
		err = knImpl.CreateLogPod(namespace, applicationName, []string{fmt.Sprintf(`while true; do echo '%s'; sleep %d; done`, message, intervalInSecond)}, podLabels, isPsaEnabled)

		if err != nil {
			println(err.Error())
			return
		}
	},
}
