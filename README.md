# Manifest

Manifest is a Go application that is designed to lint pull requests and diffs
using configurable rules. It is language agnostic, passing the relevant pull
request and diff information to scripts via JSON while using the resulting
stdout JSON to comment on the PR/diffs, fail the build, etc.

## Why use Manifest over writing individual GitHub actions?

Manifest isn't a replacement for GitHub actions, but was designed to be used
_with_ actions. It offers several benefits over using actions directly:

- Consuming standard PR information (title, description, sha, etc.) is as easy as `JSON.parse`.
- Common usage patterns like commenting on PRs or enforcing rules with good feedback is as simple as configuring Manifest and outputting JSON to stdout.
- Additional complexity like GitHub comment de-duplication and diffing PRs is handled for you.
- Unlike actions, it runs locally on any engineers machine.


## Installing Manifest

Run `go install github.com/blakewilliams/manifest/cmd/manifest` or clone+build from source.

A docker container is also provided that can be used to run manifest:

```
$ docker run -it -v $(pwd):/app -e MANIFEST_GITHUB_TOKEN=$MANIFEST_GITHUB_TOKEN ghcr.io/blakewilliams/manifest/manifest:v0.0.1`
```

## Usage

The primary usage of `manifest` is via `manifest check`, which can be configured directly in the CLI (see `manifest check help`) or a configuration file in your root directory called `manifest.config.yaml`:

```yaml
# Sample YAML config
manifest:
  concurrency: 2 # How many checks to run at once
  formatter: pretty # The formatter to use
  checks: # The check scripts to run and report on
    feature_flags:
      command: "script/feature-flag-check"
    rails_job_perform:
      command: "script/job-perform-check"
```

Then you can run `git diff main | manifest check` which will run each of the provided
checks in the provided config. Arguments provided in the config can be
overridden using the CLI flags ( see `manifest check help`).

## Writing a custom checker

Manifest checks can be written in any language since they effectively accept
JSON as stdin, and output JSON in stdout so `manifest` can output it
appropriately. The following JSON is the expected format:

Stdin:

```json
{
  "pullTitle": "My change that does A, B, C",
  "pullDescription": "A simple description of my changes",
  "repoOwner": "BlakeWilliams",
  "repoName": "manifest",
  "pullNumber": 2,
  "strict": false,
  "diff": {
    "changed": ["app/jobs/greeter_job.rb"],
    "deleted": [],
    "renamed": [],
    "new": [],
    "copied": [],
    "files": {
      "app/jobs/greeter_job.rb": {
        "operation": "change",
        "new_name": "app/jobs/greeter_job.rb",
        "old_name": "app/jobs/greeter_job.rb",
        "left": [
          {
            "lineno": 4,
            "content": "  def perform\n"
          }
        ],
        "right": [
          {
            "lineno": 4,
            "content": "  def perform(name)\n"
          }
        ]
      }
    }
  }
}
```

Stdout:

```json
{
  "failure": "",
  "comments": [
    {
      "file": "app/jobs/greeter_job.rb",
      "line": 4,
      "text": "You have modified an ActiveRecord job's arguments. In order to avoid job failures please read and follow X documentation.",
      "severity": "Warn"
    }
  ]
}
```

Comments are then output to stdout or posted to Pull Requests. The format of comments should be:

```json
{
  "file": "app/jobs/greeter_job.rb", // optional file, missing file+line comments top-level
  "line": 4, // optional line number
  "text": "don't do that because...!", // The text to output
  "severity": "Warn" // The severity of the violation. Can be one of Info, Warn, or Error.
}
```

See also the `Result` struct in `result.go` for more details on the expected output format and the `Import` struct in `manifest.go` for the expected inputs.

### Getting import JSON to test scripts

Since manifest checks work primarily through piping stdin and stdout, you'll need to generate the relevant JSON to pass to scripts utilizing `manifest`. To get JSON usable for testing or running manifest checks, you can pass `--only-import-json` to bypass running the configured scripts and return only the import JSON that would be passed to the checks.

```sh
$ cat my.diff | manifest check --json-only
```

Which should result in output like the example in "Writing a custom manifest checker". You can then pipe the JSON directly into your check script:

```sh
$ cat my.diff | manifest check --json-only | my-check
```
