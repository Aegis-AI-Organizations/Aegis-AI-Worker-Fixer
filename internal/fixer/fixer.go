package fixer

import (
	"fmt"
	"net/http"
)

type ActivitySet struct {
	Remediation RemediationActivity
}

func NewActivitySet(scanner Scanner) (ActivitySet, error) {
	prProvider, err := NewGitProviderFromEnv(http.DefaultClient)
	if err != nil {
		return ActivitySet{}, err
	}
	return ActivitySet{
		Remediation: RemediationActivity{
			Git:     &LocalGitProvider{PRProvider: prProvider},
			Scanner: scanner,
		},
	}, nil
}

func Start() {
	fmt.Println("Aegis AI Worker Fixer started. Register Temporal activity: RemediateVulnerability")
}
