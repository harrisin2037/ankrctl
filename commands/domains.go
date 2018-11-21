/*
Copyright 2018 The Dccncli Authors All rights reserved.
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

package commands

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/Ankr-network/dccn-cli"
	"github.com/Ankr-network/dccn-cli/commands/displayers"
	"github.com/Ankr-network/dccn-cli/do"
	"github.com/Ankr-network/godo"
	"github.com/spf13/cobra"
)

// Domain creates the domain commands heirarchy.
func Domain() *Command {
	cmd := &Command{
		Command: &cobra.Command{
			Use:   "domain",
			Short: "domain commands",
			Long:  "domain is used to access domain commands",
		},
		DocCategories: []string{"domain"},
		IsIndex:       true,
	}

	cmdDomainCreate := CmdBuilder(cmd, RunDomainCreate, "create <domain>", "create domain", Writer,
		aliasOpt("c"), displayerType(&displayers.Domain{}), docCategories("domain"))
	AddStringFlag(cmdDomainCreate, dccncli.ArgIPAddress, "", "", "IP address", requiredOpt())

	CmdBuilder(cmd, RunDomainList, "list", "list domains", Writer,
		aliasOpt("ls"), displayerType(&displayers.Domain{}), docCategories("domain"))

	CmdBuilder(cmd, RunDomainGet, "get <domain>", "get domain", Writer,
		aliasOpt("g"), displayerType(&displayers.Domain{}), docCategories("domain"))

	cmdRunDomainDelete := CmdBuilder(cmd, RunDomainDelete, "delete <domain>", "delete domain", Writer, aliasOpt("g"))
	AddBoolFlag(cmdRunDomainDelete, dccncli.ArgForce, dccncli.ArgShortForce, false, "Force domain delete")

	cmdRecord := &Command{
		Command: &cobra.Command{
			Use:   "records",
			Short: "domain record commands",
			Long:  "commands for interacting with an individual domain",
		},
	}
	cmd.AddCommand(cmdRecord)

	CmdBuilder(cmdRecord, RunRecordList, "list <domain>", "list records", Writer,
		aliasOpt("ls"), displayerType(&displayers.DomainRecord{}), docCategories("domain"))

	cmdRecordCreate := CmdBuilder(cmdRecord, RunRecordCreate, "create <domain>", "create record", Writer,
		aliasOpt("c"), displayerType(&displayers.DomainRecord{}), docCategories("domain"))
	AddStringFlag(cmdRecordCreate, dccncli.ArgRecordType, "", "", "Record type")
	AddStringFlag(cmdRecordCreate, dccncli.ArgRecordName, "", "", "Record name")
	AddStringFlag(cmdRecordCreate, dccncli.ArgRecordData, "", "", "Record data")
	AddIntFlag(cmdRecordCreate, dccncli.ArgRecordPriority, "", 0, "Record priority")
	AddIntFlag(cmdRecordCreate, dccncli.ArgRecordPort, "", 0, "Record port")
	AddIntFlag(cmdRecordCreate, dccncli.ArgRecordTTL, "", 1800, "Record TTL")
	AddIntFlag(cmdRecordCreate, dccncli.ArgRecordWeight, "", 0, "Record weight")
	AddIntFlag(cmdRecordCreate, dccncli.ArgRecordFlags, "", 0, "Record flags")
	AddStringFlag(cmdRecordCreate, dccncli.ArgRecordTag, "", "", "Record tag")

	cmdRunRecordDelete := CmdBuilder(cmdRecord, RunRecordDelete, "delete <domain> <record id...>", "delete record", Writer,
		aliasOpt("d"), docCategories("domain"))
	AddBoolFlag(cmdRunRecordDelete, dccncli.ArgForce, dccncli.ArgShortForce, false, "Force record delete")

	cmdRecordUpdate := CmdBuilder(cmdRecord, RunRecordUpdate, "update <domain>", "update record", Writer,
		aliasOpt("u"), displayerType(&displayers.DomainRecord{}), docCategories("domain"))
	AddIntFlag(cmdRecordUpdate, dccncli.ArgRecordID, "", 0, "Record ID")
	AddStringFlag(cmdRecordUpdate, dccncli.ArgRecordType, "", "", "Record type")
	AddStringFlag(cmdRecordUpdate, dccncli.ArgRecordName, "", "", "Record name")
	AddStringFlag(cmdRecordUpdate, dccncli.ArgRecordData, "", "", "Record data")
	AddIntFlag(cmdRecordUpdate, dccncli.ArgRecordPriority, "", 0, "Record priority")
	AddIntFlag(cmdRecordUpdate, dccncli.ArgRecordPort, "", 0, "Record port")
	AddIntFlag(cmdRecordUpdate, dccncli.ArgRecordTTL, "", 1800, "Record TTL")
	AddIntFlag(cmdRecordUpdate, dccncli.ArgRecordWeight, "", 0, "Record weight")
	AddIntFlag(cmdRecordUpdate, dccncli.ArgRecordFlags, "", 0, "Record flags")
	AddStringFlag(cmdRecordUpdate, dccncli.ArgRecordTag, "", "", "Record tag")

	return cmd
}

// RunDomainCreate runs domain create.
func RunDomainCreate(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return dccncli.NewMissingArgsErr(c.NS)
	}
	domainName := c.Args[0]

	ds := c.Domains()

	ipAddress, err := c.Ankr.GetString(c.NS, "ip-address")
	if err != nil {
		return err
	}

	req := &godo.DomainCreateRequest{
		Name:      domainName,
		IPAddress: ipAddress,
	}

	d, err := ds.Create(req)
	if err != nil {
		return err
	}

	return c.Display(&displayers.Domain{Domains: do.Domains{*d}})
}

// RunDomainList runs domain create.
func RunDomainList(c *CmdConfig) error {

	ds := c.Domains()

	domains, err := ds.List()
	if err != nil {
		return err
	}

	item := &displayers.Domain{Domains: domains}
	return c.Display(item)
}

// RunDomainGet retrieves a domain by name.
func RunDomainGet(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return dccncli.NewMissingArgsErr(c.NS)
	}
	id := c.Args[0]

	ds := c.Domains()

	if len(id) < 1 {
		return errors.New("invalid domain name")
	}

	d, err := ds.Get(id)
	if err != nil {
		return err
	}

	item := &displayers.Domain{Domains: do.Domains{*d}}
	return c.Display(item)
}

// RunDomainDelete deletes a domain by name.
func RunDomainDelete(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return dccncli.NewMissingArgsErr(c.NS)
	}
	name := c.Args[0]

	force, err := c.Ankr.GetBool(c.NS, dccncli.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirm("delete domain") == nil {
		ds := c.Domains()

		if len(name) < 1 {
			return errors.New("invalid domain name")
		}

		err := ds.Delete(name)
		return err
	} else {
		return fmt.Errorf("operation aborted")
	}

	return nil
}

// RunRecordList list records for a domain.
func RunRecordList(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return dccncli.NewMissingArgsErr(c.NS)
	}
	name := c.Args[0]

	ds := c.Domains()

	if len(name) < 1 {
		return errors.New("domain name is missing")
	}

	list, err := ds.Records(name)
	if err != nil {
		return err
	}

	items := &displayers.DomainRecord{DomainRecords: list}
	return c.Display(items)

}

// RunRecordCreate creates a domain record.
func RunRecordCreate(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return dccncli.NewMissingArgsErr(c.NS)
	}
	name := c.Args[0]

	ds := c.Domains()

	rType, err := c.Ankr.GetString(c.NS, dccncli.ArgRecordType)
	if err != nil {
		return err
	}

	rName, err := c.Ankr.GetString(c.NS, dccncli.ArgRecordName)
	if err != nil {
		return err
	}

	rData, err := c.Ankr.GetString(c.NS, dccncli.ArgRecordData)
	if err != nil {
		return err
	}

	rPriority, err := c.Ankr.GetInt(c.NS, dccncli.ArgRecordPriority)
	if err != nil {
		return err
	}

	rPort, err := c.Ankr.GetInt(c.NS, dccncli.ArgRecordPort)
	if err != nil {
		return err
	}

	rTTL, err := c.Ankr.GetInt(c.NS, dccncli.ArgRecordTTL)
	if err != nil {
		return err
	}

	rWeight, err := c.Ankr.GetInt(c.NS, dccncli.ArgRecordWeight)
	if err != nil {
		return err
	}

	rFlags, err := c.Ankr.GetInt(c.NS, dccncli.ArgRecordFlags)
	if err != nil {
		return err
	}

	rTag, err := c.Ankr.GetString(c.NS, dccncli.ArgRecordTag)
	if err != nil {
		return err
	}

	drcr := &godo.DomainRecordEditRequest{
		Type:     rType,
		Name:     rName,
		Data:     rData,
		Priority: rPriority,
		Port:     rPort,
		TTL:      rTTL,
		Weight:   rWeight,
		Flags:    rFlags,
		Tag:      rTag,
	}

	if len(drcr.Type) == 0 {
		return errors.New("record request is missing type")
	}

	r, err := ds.CreateRecord(name, drcr)
	if err != nil {
		return err
	}

	item := &displayers.DomainRecord{DomainRecords: do.DomainRecords{*r}}
	return c.Display(item)

}

// RunRecordDelete deletes a domain record.
func RunRecordDelete(c *CmdConfig) error {
	if len(c.Args) < 2 {
		return dccncli.NewMissingArgsErr(c.NS)
	}

	force, err := c.Ankr.GetBool(c.NS, dccncli.ArgForce)
	if err != nil {
		return err
	}

	if force || AskForConfirm("delete record(s)") == nil {
		domainName, ids := c.Args[0], c.Args[1:]
		if len(ids) < 1 {
			return dccncli.NewMissingArgsErr(c.NS)
		}

		ds := c.Domains()

		for _, i := range ids {
			id, err := strconv.Atoi(i)
			if err != nil {
				return fmt.Errorf("invalid record id %q", i)
			}

			err = ds.DeleteRecord(domainName, id)
			if err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("opertaion aborted")
	}

	return nil

}

// RunRecordUpdate updates a domain record.
func RunRecordUpdate(c *CmdConfig) error {
	if len(c.Args) != 1 {
		return dccncli.NewMissingArgsErr(c.NS)
	}
	domainName := c.Args[0]

	ds := c.Domains()

	recordID, err := c.Ankr.GetInt(c.NS, dccncli.ArgRecordID)
	if err != nil {
		return err
	}

	rType, err := c.Ankr.GetString(c.NS, dccncli.ArgRecordType)
	if err != nil {
		return err
	}

	rName, err := c.Ankr.GetString(c.NS, dccncli.ArgRecordName)
	if err != nil {
		return err
	}

	rData, err := c.Ankr.GetString(c.NS, dccncli.ArgRecordData)
	if err != nil {
		return err
	}

	rPriority, err := c.Ankr.GetInt(c.NS, dccncli.ArgRecordPriority)
	if err != nil {
		return err
	}

	rPort, err := c.Ankr.GetInt(c.NS, dccncli.ArgRecordPort)
	if err != nil {
		return err
	}

	rTTL, err := c.Ankr.GetInt(c.NS, dccncli.ArgRecordTTL)
	if err != nil {
		return err
	}

	rWeight, err := c.Ankr.GetInt(c.NS, dccncli.ArgRecordWeight)
	if err != nil {
		return err
	}

	rFlags, err := c.Ankr.GetInt(c.NS, dccncli.ArgRecordFlags)
	if err != nil {
		return err
	}

	rTag, err := c.Ankr.GetString(c.NS, dccncli.ArgRecordTag)
	if err != nil {
		return err
	}

	drcr := &godo.DomainRecordEditRequest{
		Type:     rType,
		Name:     rName,
		Data:     rData,
		Priority: rPriority,
		Port:     rPort,
		TTL:      rTTL,
		Weight:   rWeight,
		Flags:    rFlags,
		Tag:      rTag,
	}

	r, err := ds.EditRecord(domainName, recordID, drcr)
	if err != nil {
		return err
	}

	item := &displayers.DomainRecord{DomainRecords: do.DomainRecords{*r}}
	return c.Display(item)
}
