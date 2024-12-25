package manifest

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type noopFormatter struct{}

func (f noopFormatter) Format(checker string, i *Import, r Result) error { return nil }

//go:embed testconfig.yaml
var testConfig string

func TestConfig(t *testing.T) {
	config := &Configuration{}
	err := ParseConfig(strings.NewReader(testConfig), config, map[string]Formatter{"pretty": noopFormatter{}})
	require.NoError(t, err)

	require.Equal(t, 2, config.Concurrency)
	require.NotNil(t, config.Formatter)
	require.Len(t, config.Checkers, 1, "expected 1 plugin to be configured")
	railsJobCheck := config.Checkers["rails_job_perform"]
	require.Equal(t, "manifest checker rails_job_perform", railsJobCheck)
}
