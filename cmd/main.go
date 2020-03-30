// Copyright 2020 rvazquez@redhat.com
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package main

import (
	"github.com/roivaz/marin3r/pkg/controller"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "marin3r",
	Short: "marin3r, the simple envoy control plane",
	Run: func(cmd *cobra.Command, args []string) {
		if err := controller.NewController(); err != nil {
			panic(err)
		}
	},
}

func main() {
	rootCmd.Execute()
}
