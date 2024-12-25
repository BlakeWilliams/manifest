package checkers

import (
	"strings"
	"testing"

	"github.com/blakewilliams/manifest"
	"github.com/stretchr/testify/require"
)

var argumentChangeDiff = `
diff --git a/app/jobs/greeter_job.rb b/app/jobs/greeter_job.rb
index abc1234..def5678 100644
--- a/app/jobs/greeter_job.rb
+++ b/app/jobs/greeter_job.rb
@@ -1,7 +1,7 @@
 class GreeterJob < ApplicationJob
   queue_as :default

-  def perform
+  def perform(name)
     # Job logic here
   end
 end`

func TestManifest_NewFile(t *testing.T) {
	diff, err := manifest.NewDiff(strings.NewReader(argumentChangeDiff))
	require.NoError(t, err)

	entry := &manifest.Import{Diff: diff}
	result := &manifest.Result{Comments: make([]manifest.Comment, 0)}

	err = RailsJobArguments(entry, result)
	require.NoError(t, err)

	require.Len(t, result.Comments, 1)
	comment := result.Comments[0]

	require.Equal(t, "app/jobs/greeter_job.rb", comment.File)
	require.Equal(t, uint(4), comment.Line)
	require.Equal(t, manifest.SeverityWarn, comment.Severity)
}
