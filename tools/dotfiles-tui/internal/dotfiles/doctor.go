package dotfiles

// Severity identifies a Doctor finding's impact.
type Severity string

const (
	SeverityOK      Severity = "ok"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Finding is one deterministic, read-only Doctor result.
type Finding struct {
	Severity Severity
	Package  string
	Code     string
	Detail   string
}

// LookPath resolves one declared executable.
type LookPath func(string) (string, error)

// Doctor inspects sources, links, commands, and Git availability.
func Doctor(repo, home string, packages []Package, lookPath LookPath) []Finding {
	var findings []Finding
	if _, err := lookPath("git"); err != nil {
		findings = append(findings, Finding{SeverityWarning, "git", "command-missing", "git"})
	} else {
		findings = append(findings, Finding{SeverityOK, "git", "command-present", "git"})
	}

	for _, pkg := range packages {
		inspection, err := InspectPackage(repo, home, pkg)
		if err != nil {
			findings = append(findings, Finding{SeverityError, pkg.Name, "source-invalid", err.Error()})
		} else {
			severity := SeverityWarning
			if inspection.Status == Installed {
				severity = SeverityOK
			}
			findings = append(findings, Finding{severity, pkg.Name, "link-status", string(inspection.Status)})
		}

		if pkg.Command != "" {
			if _, err := lookPath(pkg.Command); err != nil {
				findings = append(findings, Finding{SeverityWarning, pkg.Name, "command-missing", pkg.Command})
			} else {
				findings = append(findings, Finding{SeverityOK, pkg.Name, "command-present", pkg.Command})
			}
		}
	}
	return findings
}
