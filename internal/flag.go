package internal

import "flag"

var HPClient *string
var HPServer *string
var ReportInterval *int
var PollInterval *int

func ParseArgsClient() {
	HPClient = flag.String("a", ":8080", "host and port in format <host>:<port>")

	ReportInterval = flag.Int(
		"r", 10, "period of time in seconds, throw of it will be report to the server",
	)
	PollInterval = flag.Int(
		"p", 2, "period of time in seconds, throw of it metrics will be updated from 'runtime'",
	)
	flag.Parse()
}

func ParseArgsServer() {
	HPServer = flag.String("a", ":8080", "host and port in format <host>:<port>")
	flag.Parse()
}
