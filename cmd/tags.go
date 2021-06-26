package main

/*
 * AWS SSO CLI
 * Copyright (c) 2021 Aaron Turner  <synfinatic at gmail dot com>
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
)

type TagsCmd struct {
}

func (cc *TagsCmd) Run(ctx *RunContext) error {
	sso := ctx.Config.SSO[ctx.Cli.SSO]
	sso.Refresh()

	for _, r := range sso.GetRoles() {
		fmt.Printf("%s\n", r.ARN)
		for k, v := range r.GetAllTags() {
			fmt.Printf("  %s: %s\n", k, v)
		}
		fmt.Printf("\n")
	}
	return nil
}