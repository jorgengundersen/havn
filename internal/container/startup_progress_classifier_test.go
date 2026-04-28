package container

import "testing"

func TestClassifyStartupProgress_ActivityMapping(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		activity startupProgressActivity
	}{
		{name: "fetch activity", line: "fetching store paths", activity: startupProgressActivityFetch},
		{name: "build activity", line: "building derivations 7/42", activity: startupProgressActivityBuild},
		{name: "evaluate activity", line: "resolving flake inputs", activity: startupProgressActivityEvaluate},
		{name: "other activity", line: "checking cache metadata", activity: startupProgressActivityOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyStartupProgress(tt.line)
			if got.Activity != tt.activity {
				t.Fatalf("activity = %q, want %q", got.Activity, tt.activity)
			}
		})
	}
}

func TestRenderStartupProgress_IncludesParsedMetrics(t *testing.T) {
	classified := classifyStartupProgress("fetching store paths 23/118 (1.2GiB/4.8GiB @ 18MiB/s)")

	got := renderStartupProgress(classified)
	want := "fetch: fetching store paths 23/118 (1.2GiB/4.8GiB @ 18MiB/s) (23/118, 1.2GiB/4.8GiB, @ 18MiB/s)"
	if got != want {
		t.Fatalf("rendered progress = %q, want %q", got, want)
	}
}
