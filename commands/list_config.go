// Copyright 2015 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.Print the version number of Hug

package commands

import (
	"reflect"
	"sort"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print the site configuration",
	Long:  `Print the site configuration, both default and custom settings.`,
}

func init() {
	configCmd.Flags().StringVarP(&theme, "theme", "t", "", "theme to use (located in /themes/THEMENAME/)")
	configCmd.Flags().StringVarP(&themesDir, "themesDir", "", "", "filesystem path to themes directory")

	configCmd.RunE = printConfig
}

func printConfig(cmd *cobra.Command, args []string) error {
	if cmd != configCmd {
		panic("not expected")
	}
	cfg, err := InitializeConfig(false, nil, configCmd)

	if err != nil {
		return err
	}

	jww.FEEDBACK.Printf("------------\nfinal state\n----\n%s", cfg.Cfg.(*viper.Viper).Spew())

	allSettings := cfg.Cfg.(*viper.Viper).AllSettings()
	allOrigins := cfg.Cfg.(*viper.Viper).AllSettingsLevels()

	// Keys set by flags
	var keysOverride []string
	var keysFlags []string
	var keysConfig []string
	var keysDefault []string

	for k := range allOrigins {
		switch allOrigins[k] {
		case "override":
			keysOverride = append(keysFlags, k)
		case "flag":
			keysFlags = append(keysFlags, k)
		case "config":
			keysConfig = append(keysConfig, k)
		default:
			keysDefault = append(keysDefault, k)
		}
	}
	fn := func(tag string, keys []string) {
		if len(keys) > 0 {
			jww.FEEDBACK.Printf("config from %s:\n", tag)
			sort.Strings(keys)
			for _, v := range keys {
				jww.FEEDBACK.Printf("  %s\n", v)
			}
		}
	}
	fn("override", keysOverride)
	fn("flags", keysFlags)
	fn("config", keysConfig)
	fn("default", keysDefault)

	for i, v := range allOrigins {
		jww.FEEDBACK.Printf("%s = %s\n", i, v)
	}

	var separator string
	if allSettings["metadataformat"] == "toml" {
		separator = " = "
	} else {
		separator = ": "
	}

	// Put keys in sorted order (people like it better that way)
	var keys []string
	for k := range allSettings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		tag := ""
		switch allOrigins[k] {
		case "flag":
			tag = "*"
		case "config":
			tag = "!"
		}
		kv := reflect.ValueOf(allSettings[k])
		if kv.Kind() == reflect.String {
			jww.FEEDBACK.Printf("%s%s%s\"%+v\"\n", tag, k, separator, allSettings[k])
		} else {
			jww.FEEDBACK.Printf("%s%s%s%+v\n", tag, k, separator, allSettings[k])
		}
	}

	return nil
}
