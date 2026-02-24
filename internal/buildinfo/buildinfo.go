package buildinfo

import "fmt"

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

type Info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func Current() Info {
	return Info{
		Name:    "cfgstack",
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	}
}

func (i Info) String() string {
	return fmt.Sprintf("%s %s (commit=%s, date=%s)", i.Name, i.Version, i.Commit, i.Date)
}

func String() string {
	return Current().String()
}
