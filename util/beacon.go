package util

import "os"

type Beacon struct {
	Name      string
	Target    string
	Hostname  string
	Location  string
	Platform  string
	Executors []string
	Range     string
	Sleep     int
	Pwd       string
	Executing string
	Links     []Instruction
}

// RefreshBeacon updates the Beacon with the values from the AgentConfig.
func RefreshBeacon(agent *AgentConfig, beacon *Beacon, contact string) {
	pwd, _ := os.Getwd()
	beacon.Sleep = agent.Sleep
	beacon.Range = agent.Range
	beacon.Pwd = pwd
	beacon.Target = agent.Contact[contact]
	beacon.Executing = agent.BuildExecutingHash()
}
