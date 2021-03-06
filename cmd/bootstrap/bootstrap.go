// Copyright © 2019 Sergey Nuzhdin ipaq.lw@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bootstrap

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lwolf/kube-atlas/pkg/bootstrap"
)

var (
	dir         string
	interactive bool
)

var initUsage = `Init command
`

// initCmd represents the init command
var CmdInit = &cobra.Command{
	Use:   "init",
	Short: "Create a new kube-atlas.yaml file in the current directory",
	Long:  initUsage,
	Run: func(cmd *cobra.Command, args []string) {
		log.Warn().Msg("init command is not yet implemented")
		if interactive {
			bootstrap.Interactive()
		}
		// else {
		// bootstrap.Execute()
		// }
	},
}

func init() {

	// Here you will define your flags and configuration settings.
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	CmdInit.Flags().BoolVar(&interactive, "interactive", false, "Start in interactive mode")
}
