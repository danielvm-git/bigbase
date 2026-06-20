package mcp

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed knowledge/services.json
var servicesJSON []byte

//go:embed knowledge/frameworks.json
var frameworksJSON []byte

//go:embed knowledge/examples/code-examples.json
var codeExamplesJSON []byte

// servicesData mirrors knowledge/services.json structure.
type servicesData struct {
	Services []serviceEntry `json:"services"`
}

type serviceEntry struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
	Status       string   `json:"status"`
	Endpoints    []string `json:"endpoints"`
}

// frameworksData mirrors knowledge/frameworks.json structure.
type frameworksData struct {
	Frameworks []frameworkEntry `json:"frameworks"`
}

type frameworkEntry struct {
	Name           string `json:"name"`
	Display        string `json:"display"`
	Maturity       string `json:"maturity"`
	RecommendedFor string `json:"recommended_for"`
	Caveats        string `json:"caveats"`
}

// codeExamplesData mirrors knowledge/examples/code-examples.json.
type codeExamplesData struct {
	Examples []exampleEntry `json:"examples"`
}

type exampleEntry struct {
	Service   string `json:"service"`
	Framework string `json:"framework"`
	Title     string `json:"title"`
	Code      string `json:"code"`
}

func loadServices() ([]serviceEntry, error) {
	var data servicesData
	if err := json.Unmarshal(servicesJSON, &data); err != nil {
		return nil, err
	}
	return data.Services, nil
}

func loadFrameworks() ([]frameworkEntry, error) {
	var data frameworksData
	if err := json.Unmarshal(frameworksJSON, &data); err != nil {
		return nil, err
	}
	return data.Frameworks, nil
}

func loadCodeExamples() ([]exampleEntry, error) {
	var data codeExamplesData
	if err := json.Unmarshal(codeExamplesJSON, &data); err != nil {
		return nil, err
	}
	return data.Examples, nil
}

func formatServicesList(services []serviceEntry) string {
	var b strings.Builder
	b.WriteString("# BigBase Services\n\n")
	for _, s := range services {
		fmt.Fprintf(&b, "## %s (%s)\n", s.Name, s.Status)
		fmt.Fprintf(&b, "%s\n\n", s.Description)
		b.WriteString("**Capabilities:** ")
		b.WriteString(strings.Join(s.Capabilities, ", "))
		b.WriteString("\n\n")
	}
	return b.String()
}

func formatServiceDoc(svc serviceEntry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", svc.Name)
	fmt.Fprintf(&b, "**Status:** %s\n\n", svc.Status)
	fmt.Fprintf(&b, "%s\n\n", svc.Description)
	b.WriteString("## Capabilities\n\n")
	for _, c := range svc.Capabilities {
		fmt.Fprintf(&b, "- %s\n", c)
	}
	b.WriteString("\n## Endpoints\n\n")
	for _, e := range svc.Endpoints {
		fmt.Fprintf(&b, "- `%s`\n", e)
	}
	return b.String()
}

func formatFrameworksList(frameworks []frameworkEntry) string {
	var b strings.Builder
	b.WriteString("# Supported Frameworks\n\n")
	for _, f := range frameworks {
		fmt.Fprintf(&b, "## %s\n", f.Display)
		fmt.Fprintf(&b, "**Maturity:** %s\n", f.Maturity)
		fmt.Fprintf(&b, "**Best for:** %s\n", f.RecommendedFor)
		if f.Caveats != "" {
			fmt.Fprintf(&b, "**Note:** %s\n", f.Caveats)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func formatCodeExample(ex exampleEntry) string {
	return fmt.Sprintf("# %s\n\n**Service:** %s | **Framework:** %s\n\n```%s\n%s\n```\n",
		ex.Title, ex.Service, ex.Framework, langForFramework(ex.Framework), ex.Code)
}

func langForFramework(fw string) string {
	switch fw {
	case "sveltekit", "react", "nextjs", "vue":
		return "typescript"
	case "go":
		return "go"
	case "python":
		return "python"
	default:
		return "javascript"
	}
}
