/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provision
/*
import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/inspur-incloud/cloud-provider-ics/pkg/cli"
)

var (
	configFile      string
	interactive     bool


	// iCenter IP.
	ichost string
	// vCenter port.
	icport string

	// iCenter username.
	icUser string
	// iCenter password in clear text.
	icPassword string

	// vcRole is role for solution user (Default is admin)
	icRole string
)

var provisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Initialize provisioning with ics cloud provider",
	Long: `Starting prerequisites for deploying a cloud provider on ics, in cluding :
	[x] ics configuration health check.
	[x] Create ics solution user.
	[x] Create ics role with minimal set of permissions.
  `,
	Example: `# Specify interaction mode or declaration mode (default)
	icsctl provision --interactive=false
`,
	Run: RunProvision,
}

// AddProvision initializes the "provision" command.
func AddProvision(cmd *cobra.Command) {

	provisionCmd.Flags().StringVar(&configFile, "config", "", "ics cloud provider config file path")

	provisionCmd.Flags().BoolVar(&interactive, "interactive", true, "Specify interactive mode (true) as default, set (false) for declarative mode for automation")

	provisionCmd.Flags().StringVar(&ichost, "host", "", "Specify iCenter IP")
	provisionCmd.Flags().StringVar(&icport, "port", "", "Specify iCenter Port")
	provisionCmd.Flags().StringVar(&icUser, "user", "", "Specify iCenter user")
	provisionCmd.Flags().StringVar(&icPassword, "password", "", "Specify iCenter Password")


	provisionCmd.Flags().StringVar(&icRole, "role", "Administrator", "Role for solution user (RegularUser|Administrator)")

	cmd.AddCommand(provisionCmd)
}

// RunProvision executes the "provision" command.
func RunProvision(cmd *cobra.Command, args []string) {
	// TODO (fanz): implement provision
	fmt.Println("Perform cloud provider provisioning...")
	o := cli.ClientOption{}
	o.LoadCredential(icUser, icPassword, icRole)
	ctx := context.Background()
	client, err := o.NewClient(ctx, ichost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer client.Logout(ctx)
	o.Client = client
	fmt.Println("Create solution user...")
	err = cli.CreateSolutionUser(ctx, &o)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	r := cli.Role{RoleName: "k8s-ics-default"}
	fmt.Println("Create default role with minimal permissions...")
	err = cli.CreateRole(ctx, &o, &r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Checking ics Config on VMs...")
	err = cli.CheckICsConfig(ctx, &o)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
*/