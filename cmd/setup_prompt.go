package main

/*
 * AWS SSO CLI
 * Copyright (c) 2021-2022 Aaron Turner  <synfinatic at gmail dot com>
 *
 * This program is free software: you can redistribute it
 * and/or modify it under the terms of the GNU General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or with the authors permission any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"runtime"
	"strconv"

	"github.com/manifoldco/promptui"
	"github.com/synfinatic/aws-sso-cli/internal/utils"
)

const (
	START_URL_FORMAT  = "https://%s.awsapps.com/start"
	START_FQDN_FORMAT = "%s.awsapps.com"
)

// https://docs.aws.amazon.com/general/latest/gr/sso.html
var AvailableAwsSSORegions []string = []string{
	"us-east-1",
	"us-east-2",
	"us-west-2",
	"ap-south-1",
	"ap-northeast-2",
	"ap-southeast-1",
	"ap-southeast-2",
	"ap-northeast-1",
	"ca-central-1",
	"eu-central-1",
	"eu-west-1",
	"eu-west-2",
	"eu-west-3",
	"eu-north-1",
	"sa-east-1",
	"us-gov-west-1",
}

var ssoNameRegexp *regexp.Regexp

func promptSsoInstance(defaultValue string) string {
	var val string
	var err error

	// Name our SSO instance
	prompt := promptui.Prompt{
		Label: "SSO Instance Name (DefaultSSO)",
		Validate: func(input string) error {
			if ssoNameRegexp == nil {
				ssoNameRegexp, _ = regexp.Compile(`^[a-zA-Z0-9-_@:]+$`)
			}
			if len(input) > 0 && ssoNameRegexp.Match([]byte(input)) {
				return nil
			}
			return fmt.Errorf("SSO Name must be a valid string")
		},
		Default: defaultValue,
		Pointer: promptui.PipeCursor,
	}
	if val, err = prompt.Run(); err != nil {
		log.Fatal(err)
	}
	return val
}

var ssoHostnameRegexp *regexp.Regexp

func promptStartUrl(defaultValue string) string {
	var val string
	var err error
	validFQDN := false

	for !validFQDN {
		// Get the hostname of the AWS SSO start URL
		prompt := promptui.Prompt{
			Label: "SSO Start URL Hostname (XXXXXXX.awsapps.com)",
			Validate: func(input string) error {
				if ssoHostnameRegexp == nil {
					ssoHostnameRegexp, _ = regexp.Compile(`^([a-zA-Z0-9-]+)(\.awsapps\.com)?$`)
				}
				if len(input) > 0 && len(input) < 64 && ssoHostnameRegexp.Match([]byte(input)) {
					return nil
				}
				return fmt.Errorf("Invalid DNS hostname: %s", input)
			},
			Default: defaultValue,
			Pointer: promptui.PipeCursor,
		}
		if val, err = prompt.Run(); err != nil {
			log.Fatal(err)
		}

		if _, err := net.LookupHost(fmt.Sprintf(START_FQDN_FORMAT, val)); err == nil {
			validFQDN = true
		} else if err != nil {
			log.Errorf("Unable to resolve %s", fmt.Sprintf(START_FQDN_FORMAT, val))
		}
	}

	return val
}

func promptAwsSsoRegion(defaultValue string) string {
	var val string
	var err error

	pos := 0
	for i, v := range AvailableAwsSSORegions {
		if v == defaultValue {
			pos = i
		}
	}

	// Pick our AWS SSO region
	label := "AWS SSO Region (SSORegion)"
	sel := promptui.Select{
		Label:        label,
		Items:        AvailableAwsSSORegions,
		HideSelected: false,
		CursorPos:    pos,
		Stdout:       &utils.BellSkipper{},
		Templates: &promptui.SelectTemplates{
			Selected: fmt.Sprintf(`%s: {{ . | faint }}`, label),
		},
	}
	if _, val, err = sel.Run(); err != nil {
		log.Fatal(err)
	}

	return val
}

func promptDefaultRegion(defaultValue string) string {
	var val string
	var err error

	// Pick the default AWS region to use
	defaultRegions := []string{"None"}
	defaultRegions = append(defaultRegions, AvailableAwsRegions...)

	for _, v := range defaultRegions {
		if v == defaultValue {
			return v
		}
	}

	label := "Default region for connecting to AWS (DefaultRegion)"
	sel := promptui.Select{
		Label:        label,
		Items:        defaultRegions,
		CursorPos:    index(defaultRegions, defaultValue),
		HideSelected: false,
		Stdout:       &utils.BellSkipper{},
		Templates: &promptui.SelectTemplates{
			Selected: fmt.Sprintf(`%s: {{ . | faint }}`, label),
		},
	}
	if _, val, err = sel.Run(); err != nil {
		log.Fatal(err)
	}

	if val == "None" {
		val = ""
	}

	return val
}

// promptUseFirefox asks if the user wants to use firefox containers
// and if so, returns the path to the Firefox binary
func promptUseFirefox(defaultValue string) string {
	var val, useFirefox string
	var err error
	idx := 1

	if defaultValue != "" {
		idx = 0
	}

	label := "Use Firefox containers to open URLs?"
	sel := promptui.Select{
		Label:        label,
		HideSelected: false,
		Items:        []string{"Yes", "No"},
		CursorPos:    idx,
		Stdout:       &utils.BellSkipper{},
		Templates: &promptui.SelectTemplates{
			Selected: fmt.Sprintf(`%s: {{ . | faint }}`, label),
		},
	}
	if _, useFirefox, err = sel.Run(); err != nil {
		log.Fatal(err)
	}

	if useFirefox == "Yes" {
		fmt.Printf("Ensure that you have the 'Open external links in a container' plugin for Firefox.")
		prompt := promptui.Prompt{
			Label:    "Path to Firefox binary",
			Stdout:   &utils.BellSkipper{},
			Default:  firefoxDefaultBrowserPath(defaultValue),
			Pointer:  promptui.PipeCursor,
			Validate: validateBinary,
		}
		if val, err = prompt.Run(); err != nil {
			log.Fatal(err)
		}
	}

	return val
}

func promptUrlAction(defaultValue string) string {
	var val string
	var err error
	items := []string{"clip", "exec", "open", "print", "printurl"}

	// How should we deal with URLs?  Note we don't support `exec`
	// here since that is an "advanced" feature
	label := "Default action to take with URLs (UrlAction)"
	sel := promptui.Select{
		Label:     label,
		CursorPos: index(items, defaultValue),
		Items:     items,
		Stdout:    &utils.BellSkipper{},
		Templates: &promptui.SelectTemplates{
			Selected: fmt.Sprintf(`%s: {{ . | faint }}`, label),
		},
	}
	if _, val, err = sel.Run(); err != nil {
		log.Fatal(err)
	}

	return val
}

func promptUrlExecCommand(defaultValue []interface{}) []interface{} {
	var val []interface{}
	var err error
	var line string
	argNum := 1

	fmt.Printf("Please enter one per line, the command and list of arguments for UrlExecCommand\n")

	command := defaultValue[0].(string)
	prompt := promptui.Prompt{
		Label:    "Binary to execute to open URLs",
		Default:  command,
		Stdout:   &utils.BellSkipper{},
		Validate: validateBinary,
		Pointer:  promptui.PipeCursor,
	}

	if line, err = prompt.Run(); err != nil {
		log.Fatal(err)
	}

	val = append(val, interface{}(line))

	// zero out the defaults if we change the command to execute
	if line != defaultValue[0] {
		defaultValue = []interface{}{}
	}

	for line != "" {
		arg := ""
		if argNum < len(defaultValue) {
			arg = defaultValue[argNum].(string)
		}
		prompt = promptui.Prompt{
			Label:   fmt.Sprintf("Enter argument #%d or empty string to stop", argNum),
			Default: arg,
			Stdout:  &utils.BellSkipper{},
			Pointer: promptui.PipeCursor,
		}
		if line, err = prompt.Run(); err != nil {
			log.Fatal(err)
		}
		if line != "" {
			val = append(val, line)
		}
		argNum++
	}
	return val
}

func promptDefaultBrowser(defaultValue string) string {
	var val string
	var err error

	prompt := promptui.Prompt{
		Label:    "Specify path to browser to use. Leave empty to use system default (Browser)",
		Default:  defaultValue,
		Stdout:   &utils.BellSkipper{},
		Pointer:  promptui.PipeCursor,
		Validate: validateBinaryOrNone,
	}

	if val, err = prompt.Run(); err != nil {
		log.Fatal(err)
	}

	return val
}

func promptConsoleDuration(defaultValue int32) int32 {
	var val string
	var err error

	// https://docs.aws.amazon.com/STS/latest/APIReference/API_GetFederationToken.html
	prompt := promptui.Prompt{
		Label: "Number of minutes before AWS Console sessions expire (ConsoleDuration)",
		Validate: func(input string) error {
			x, err := strconv.ParseInt(input, 10, 64)
			if err != nil || x > 2160 || x < 15 {
				return fmt.Errorf("Value must be a valid integer between 15 and 2160")
			}
			return nil
		},
		Stdout:  &utils.BellSkipper{},
		Default: fmt.Sprintf("%d", defaultValue),
		Pointer: promptui.PipeCursor,
	}
	if val, err = prompt.Run(); err != nil {
		log.Fatal(err)
	}

	x, _ := strconv.ParseInt(val, 10, 32)
	return int32(x)
}

func promptHistoryLimit(defaultValue int64) int64 {
	var val string
	var err error

	prompt := promptui.Prompt{
		Label:    "Maximum number of History items to keep (HistoryLimit)",
		Validate: validateInteger,
		Stdout:   &utils.BellSkipper{},
		Default:  fmt.Sprintf("%d", defaultValue),
		Pointer:  promptui.PipeCursor,
	}
	if val, err = prompt.Run(); err != nil {
		log.Fatal(err)
	}

	x, _ := strconv.ParseInt(val, 10, 64)
	return x
}

func promptHistoryMinutes(defaultValue int64) int64 {
	var val string
	var err error

	prompt := promptui.Prompt{
		Label:    "Number of minutes to keep items in History (HistoryMinutes)",
		Validate: validateInteger,
		Default:  fmt.Sprintf("%d", defaultValue),
		Stdout:   &utils.BellSkipper{},
		Pointer:  promptui.PipeCursor,
	}
	if val, err = prompt.Run(); err != nil {
		log.Fatal(err)
	}

	x, _ := strconv.ParseInt(val, 10, 64)
	return x
}

func promptLogLevel(defaultValue string) string {
	var val string
	var err error

	logLevels := []string{
		"error",
		"warn",
		"info",
		"debug",
		"trace",
	}

	label := "Log Level (LogLevel)"
	sel := promptui.Select{
		Label:        label,
		Items:        logLevels,
		CursorPos:    index(logLevels, defaultValue),
		HideSelected: false,
		Stdout:       &utils.BellSkipper{},
		Templates: &promptui.SelectTemplates{
			Selected: fmt.Sprintf(`%s: {{ . | faint }}`, label),
		},
	}
	if _, val, err = sel.Run(); err != nil {
		log.Fatal(err)
	}
	return val
}

func yesNo(x bool) int {
	if x {
		return 0
	}
	return 1
}

// index returns the slice index of the value.  Useful for CursorPos
func index(s []string, v string) int {
	for i, x := range s {
		if v == x {
			return i
		}
	}
	return 0
}

func promptAutoConfigCheck(flag bool) bool {
	var val string
	var err error

	label := "Auto update AWS SSO cache? (AutoConfigCheck)"
	sel := promptui.Select{
		Label:        label,
		Items:        []string{"Yes", "No"},
		CursorPos:    yesNo(flag),
		HideSelected: false,
		Stdout:       &utils.BellSkipper{},
		Templates: &promptui.SelectTemplates{
			Selected: fmt.Sprintf(`%s: {{ . | faint }}`, label),
		},
	}
	if _, val, err = sel.Run(); err != nil {
		log.WithError(err).Fatalf("Unable to select AutoConfigCheck")
	}

	return val == "Yes"
}

func promptCacheRefresh(defaultValue int64) int64 {
	var val string
	var err error
	prompt := promptui.Prompt{
		Label:    "Hours between AWS SSO cache refresh. 0 to disable (CacheRefresh)",
		Validate: validateInteger,
		Default:  fmt.Sprintf("%d", defaultValue),
		Pointer:  promptui.PipeCursor,
	}

	if val, err = prompt.Run(); err != nil {
		log.Fatal(err)
	}
	x, _ := strconv.ParseInt(val, 10, 64)
	return x
}

func promptConfigProfilesUrlAction(defaultValue string) string {
	var val string
	var err error

	label := "How to open URLs via $AWS_PROFILE (ConfigProfilesUrlAction)"
	sel := promptui.Select{
		Label:        label,
		Items:        CONFIG_OPEN_OPTIONS,
		CursorPos:    index(CONFIG_OPEN_OPTIONS, defaultValue),
		HideSelected: false,
		Stdout:       &utils.BellSkipper{},
		Templates: &promptui.SelectTemplates{
			Selected: fmt.Sprintf(`%s: {{ . | faint }}`, label),
		},
	}

	if _, val, err = sel.Run(); err != nil {
		log.Fatal(err)
	}

	return val
}

func validateInteger(input string) error {
	_, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return fmt.Errorf("Value must be a valid integer")
	}
	return nil
}

// validateBinary ensures the input is a valid binary on the system
func validateBinary(input string) error {
	s, err := os.Stat(input)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		// Windows doesn't have file permissions
		if s.Mode().IsRegular() {
			return nil
		}
	default:
		// must be a file and user execute bit set
		if s.Mode().IsRegular() && s.Mode().Perm()&0100 > 0 {
			return nil
		}
	}
	return fmt.Errorf("not a valid valid")
}

// validateBinaryOrNone is just like validateBinary(), but we accept
// an empty string
func validateBinaryOrNone(input string) error {
	if input == "" {
		return nil
	}

	s, err := os.Stat(input)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		// Windows doesn't have file permissions
		if s.Mode().IsRegular() {
			return nil
		}
	default:
		// must be a file and user execute bit set
		if s.Mode().IsRegular() && s.Mode().Perm()&0100 > 0 {
			return nil
		}
	}
	return fmt.Errorf("not a valid valid")
}

// returns the default path to the firefox browser
func firefoxDefaultBrowserPath(path string) string {
	if len(path) != 0 {
		return path
	}

	switch runtime.GOOS {
	case "darwin":
		path = "/Applications/Firefox.app/Contents/MacOS/firefox"
	case "linux":
		path = "/usr/bin/firefox"
	case "windows":
		path = "\\Program Files\\Mozilla Firefox\\firefox.exe"
	default:
	}
	return path
}