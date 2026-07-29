package verifytoolchain

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Contract is the on-disk toolchain.toml structure.
type Contract struct {
	Meta  ContractMeta            `toml:"meta"`
	Tools ContractTools           `toml:"tools"`
	// extra catches unknown top-level keys so we can warn instead of silently
	// dropping typos in tool names.
	Extra map[string]toml.Primitive `toml:"-"`
}

type ContractMeta struct {
	Version     int    `toml:"version"`
	Description string `toml:"description"`
}

type ContractTools struct {
	Required map[string]ToolSpec `toml:"required"`
	Optional map[string]ToolSpec `toml:"optional"`
}

// ToolSpec declares one tool. Min is the semantic-version floor; an empty Min
// means "presence only" (do not enforce a version).
type ToolSpec struct {
	Min string `toml:"min"`
}

// LoadContract reads and validates toolchain.toml at path.
func LoadContract(path string) (*Contract, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read toolchain contract %s: %w", path, err)
	}
	var c Contract
	md, err := toml.Decode(string(b), &c)
	if err != nil {
		return nil, fmt.Errorf("decode toolchain contract %s: %w", path, err)
	}
	// The decoder ignores unknown keys by default; surface them as warnings
	// via Undecoded so a typo like [tools.requierd] doesn't silently pass.
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		names := make([]string, 0, len(undecoded))
		for _, k := range undecoded {
			names = append(names, k.String())
		}
		return nil, fmt.Errorf("toolchain contract %s has unknown keys (typo?): %v", path, names)
	}
	if len(c.Tools.Required) == 0 && len(c.Tools.Optional) == 0 {
		return nil, fmt.Errorf("toolchain contract %s declares no tools (empty [tools.required] and [tools.optional])", path)
	}
	// Validate that every declared min parses, so a malformed floor fails at
	// contract load rather than per-tool at verify time.
	for name, spec := range c.Tools.Required {
		if spec.Min == "" {
			continue
		}
		if _, err := parseNumeric(spec.Min); err != nil {
			return nil, fmt.Errorf("toolchain contract %s: required tool %q has invalid min %q: %w", path, name, spec.Min, err)
		}
	}
	return &c, nil
}

// RequiredTools returns the required-tool map in a deterministic order (sorted
// by name) so output is stable across runs/maps.
func (c *Contract) RequiredTools() []string {
	names := make([]string, 0, len(c.Tools.Required))
	for n := range c.Tools.Required {
		names = append(names, n)
	}
	sortStrings(names)
	return names
}

func (c *Contract) OptionalTools() []string {
	names := make([]string, 0, len(c.Tools.Optional))
	for n := range c.Tools.Optional {
		names = append(names, n)
	}
	sortStrings(names)
	return names
}
