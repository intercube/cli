/*
Copyright © 2021 Intercube <opensource@intercube.io>

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
package cmd

import (
	"errors"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/tcnksm/go-httpstat"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"
)

// speedtestCmd represents the speedtest command
var speedtestCmd = &cobra.Command{
	Use:   "speedtest [url]",
	Short: "Speed tests a site from this machine",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("The type argument is required")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		speedtest(args[0])
	},
}

func speedtest(testableUrl string) {
	t := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	c := &http.Client{
		Timeout:   time.Second * 10,
		Transport: t,
	}

	req, err := http.NewRequest("GET", testableUrl, nil)
	if err != nil {
		panic(err)
	}

	var result httpstat.Result
	ctx := httpstat.WithHTTPStat(req.Context(), &result)
	req = req.WithContext(ctx)

	resp, err := c.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)

	end := time.Now()

	speedData := [][]string{
		{"DNS lookup", fmt.Sprintf("%d ms", int(result.DNSLookup/time.Millisecond))},
		{"TCP connection", fmt.Sprintf("%d ms", int(result.TCPConnection/time.Millisecond))},
		{"TLS handshake", fmt.Sprintf("%d ms", int(result.TCPConnection/time.Millisecond))},
		{"Server processing", fmt.Sprintf("%d ms", int(result.ServerProcessing/time.Millisecond))},
		{"Content transfer", fmt.Sprintf("%d ms", int(result.ContentTransfer(end)/(time.Millisecond)))},
		{"Name Lookup", fmt.Sprintf("%d ms", int(result.NameLookup/time.Millisecond))},
		{"Connect", fmt.Sprintf("%d ms", int(result.Connect/time.Millisecond))},
		{"Pre Transfer", fmt.Sprintf("%d ms", int(result.Pretransfer/time.Millisecond))},
		{"Start Transfer", fmt.Sprintf("%d ms", int(result.StartTransfer/time.Millisecond))},
		{"Total", fmt.Sprintf("%d ms", int(result.Total(end)/time.Millisecond))},
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Result type", "Time taken"})

	for _, v := range speedData {
		table.Append(v)
	}
	table.Render()
}

func init() {
	rootCmd.AddCommand(speedtestCmd)
}
