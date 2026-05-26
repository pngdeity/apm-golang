package install

import (
	"strings"
	"testing"
)

// TestParityErrors verifies that the install error types satisfy the error
// interface and carry their fields correctly -- mirroring errors.py.
func TestParityErrors(t *testing.T) {
	t.Run("DirectDependencyError", func(t *testing.T) {
		err := NewDirectDependencyError("dep foo failed")
		if err.Error() != "dep foo failed" {
			t.Fatalf("unexpected message: %q", err.Error())
		}
	})

	t.Run("AuthenticationError_message", func(t *testing.T) {
		err := NewAuthenticationError("auth failed for github.com", "run: gh auth login")
		if err.Error() != "auth failed for github.com" {
			t.Fatalf("unexpected message: %q", err.Error())
		}
		if err.DiagnosticContext != "run: gh auth login" {
			t.Fatalf("unexpected context: %q", err.DiagnosticContext)
		}
	})

	t.Run("AuthenticationError_empty_context", func(t *testing.T) {
		err := NewAuthenticationError("no token", "")
		if err.DiagnosticContext != "" {
			t.Fatalf("expected empty context, got %q", err.DiagnosticContext)
		}
	})

	t.Run("FrozenInstallError_no_reasons", func(t *testing.T) {
		err := NewFrozenInstallError("lockfile missing", nil)
		if err.Error() != "lockfile missing" {
			t.Fatalf("unexpected message: %q", err.Error())
		}
		if len(err.Reasons) != 0 {
			t.Fatalf("expected empty reasons, got %v", err.Reasons)
		}
	})

	t.Run("FrozenInstallError_with_reasons", func(t *testing.T) {
		reasons := []string{"dep a missing", "dep b missing"}
		err := NewFrozenInstallError("lockfile stale", reasons)
		if len(err.Reasons) != 2 {
			t.Fatalf("expected 2 reasons, got %d", len(err.Reasons))
		}
		if err.Reasons[0] != "dep a missing" {
			t.Fatalf("unexpected reason: %q", err.Reasons[0])
		}
	})

	t.Run("PolicyViolationError", func(t *testing.T) {
		err := NewPolicyViolationError("policy blocked install", "org:acme/.github")
		if err.Error() != "policy blocked install" {
			t.Fatalf("unexpected message: %q", err.Error())
		}
		if err.PolicySource != "org:acme/.github" {
			t.Fatalf("unexpected source: %q", err.PolicySource)
		}
	})
}

// TestParityPlanEntry verifies PlanEntry computed properties -- mirroring plan.py.
func TestParityPlanEntry(t *testing.T) {
	t.Run("HasChanges_update", func(t *testing.T) {
		e := PlanEntry{DepKey: "foo", Action: ActionUpdate}
		if !e.HasChanges() {
			t.Fatal("update entry should have changes")
		}
	})

	t.Run("HasChanges_unchanged", func(t *testing.T) {
		e := PlanEntry{DepKey: "foo", Action: ActionUnchanged}
		if e.HasChanges() {
			t.Fatal("unchanged entry should not have changes")
		}
	})

	t.Run("ShortOldCommit_full_sha", func(t *testing.T) {
		e := PlanEntry{OldResolvedCommit: "abcdef1234567890"}
		if e.ShortOldCommit() != "abcdef1" {
			t.Fatalf("expected 7-char sha, got %q", e.ShortOldCommit())
		}
	})

	t.Run("ShortOldCommit_empty", func(t *testing.T) {
		e := PlanEntry{}
		if e.ShortOldCommit() != "-" {
			t.Fatalf("expected '-', got %q", e.ShortOldCommit())
		}
	})

	t.Run("ShortNewCommit", func(t *testing.T) {
		e := PlanEntry{NewResolvedCommit: "deadbeef0123456"}
		if e.ShortNewCommit() != "deadbee" {
			t.Fatalf("expected 'deadbee', got %q", e.ShortNewCommit())
		}
	})
}

// TestParityUpdatePlan verifies UpdatePlan methods -- mirroring plan.py.
func TestParityUpdatePlan(t *testing.T) {
	t.Run("HasChanges_empty", func(t *testing.T) {
		p := UpdatePlan{}
		if p.HasChanges() {
			t.Fatal("empty plan should not have changes")
		}
	})

	t.Run("HasChanges_all_unchanged", func(t *testing.T) {
		p := UpdatePlan{Entries: []PlanEntry{
			{Action: ActionUnchanged},
			{Action: ActionUnchanged},
		}}
		if p.HasChanges() {
			t.Fatal("all-unchanged plan should not have changes")
		}
	})

	t.Run("HasChanges_with_update", func(t *testing.T) {
		p := UpdatePlan{Entries: []PlanEntry{
			{Action: ActionUnchanged},
			{Action: ActionUpdate},
		}}
		if !p.HasChanges() {
			t.Fatal("plan with update should have changes")
		}
	})

	t.Run("ChangedEntries", func(t *testing.T) {
		p := UpdatePlan{Entries: []PlanEntry{
			{DepKey: "a", Action: ActionUnchanged},
			{DepKey: "b", Action: ActionUpdate},
			{DepKey: "c", Action: ActionAdd},
		}}
		changed := p.ChangedEntries()
		if len(changed) != 2 {
			t.Fatalf("expected 2 changed entries, got %d", len(changed))
		}
	})

	t.Run("SummaryCounts", func(t *testing.T) {
		p := UpdatePlan{Entries: []PlanEntry{
			{Action: ActionUpdate},
			{Action: ActionUpdate},
			{Action: ActionAdd},
			{Action: ActionRemove},
			{Action: ActionUnchanged},
		}}
		counts := p.SummaryCounts()
		if counts[ActionUpdate] != 2 {
			t.Fatalf("expected 2 updates, got %d", counts[ActionUpdate])
		}
		if counts[ActionAdd] != 1 {
			t.Fatalf("expected 1 add, got %d", counts[ActionAdd])
		}
		if counts[ActionRemove] != 1 {
			t.Fatalf("expected 1 remove, got %d", counts[ActionRemove])
		}
		if counts[ActionUnchanged] != 1 {
			t.Fatalf("expected 1 unchanged, got %d", counts[ActionUnchanged])
		}
	})
}

// TestParityBuildUpdatePlan verifies the plan diff logic -- mirroring build_update_plan.
func TestParityBuildUpdatePlan(t *testing.T) {
	t.Run("all_new_deps", func(t *testing.T) {
		resolved := []ResolvedDep{
			{Key: "github.com/org/a", DisplayName: "a", ResolvedRef: "main", ResolvedCommit: "abc1234"},
			{Key: "github.com/org/b", DisplayName: "b", ResolvedRef: "v1.0", ResolvedCommit: "def5678"},
		}
		plan := BuildUpdatePlan(nil, resolved)
		if len(plan.Entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(plan.Entries))
		}
		for _, e := range plan.Entries {
			if e.Action != ActionAdd {
				t.Fatalf("expected all adds, got %q", e.Action)
			}
		}
	})

	t.Run("unchanged_dep", func(t *testing.T) {
		old := map[string]LockfileEntry{
			"github.com/org/a": {RepoURL: "github.com/org/a", ResolvedRef: "main", ResolvedCommit: "abc1234"},
		}
		resolved := []ResolvedDep{
			{Key: "github.com/org/a", ResolvedRef: "main", ResolvedCommit: "abc1234"},
		}
		plan := BuildUpdatePlan(old, resolved)
		if len(plan.Entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(plan.Entries))
		}
		if plan.Entries[0].Action != ActionUnchanged {
			t.Fatalf("expected unchanged, got %q", plan.Entries[0].Action)
		}
		if plan.HasChanges() {
			t.Fatal("plan with only unchanged entries should not have changes")
		}
	})

	t.Run("updated_dep", func(t *testing.T) {
		old := map[string]LockfileEntry{
			"github.com/org/a": {RepoURL: "github.com/org/a", ResolvedRef: "main", ResolvedCommit: "abc1234"},
		}
		resolved := []ResolvedDep{
			{Key: "github.com/org/a", ResolvedRef: "main", ResolvedCommit: "xyz9999"},
		}
		plan := BuildUpdatePlan(old, resolved)
		if plan.Entries[0].Action != ActionUpdate {
			t.Fatalf("expected update, got %q", plan.Entries[0].Action)
		}
	})

	t.Run("removed_dep", func(t *testing.T) {
		old := map[string]LockfileEntry{
			"github.com/org/a": {RepoURL: "github.com/org/a", ResolvedRef: "main", ResolvedCommit: "abc1234"},
			"github.com/org/b": {RepoURL: "github.com/org/b", ResolvedRef: "v1", ResolvedCommit: "def5678"},
		}
		resolved := []ResolvedDep{
			{Key: "github.com/org/a", ResolvedRef: "main", ResolvedCommit: "abc1234"},
		}
		plan := BuildUpdatePlan(old, resolved)
		if len(plan.Entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(plan.Entries))
		}
		var hasRemove bool
		for _, e := range plan.Entries {
			if e.Action == ActionRemove {
				hasRemove = true
			}
		}
		if !hasRemove {
			t.Fatal("expected a remove entry for missing dep")
		}
	})
}

// TestParityLockfileSatisfiesManifest verifies the frozen-install satisfaction check.
func TestParityLockfileSatisfiesManifest(t *testing.T) {
	t.Run("satisfied", func(t *testing.T) {
		locked := map[string]bool{"github.com/org/a": true, "github.com/org/b": true}
		deps := []ManifestDep{{Key: "github.com/org/a"}, {Key: "github.com/org/b"}}
		ok, reasons := LockfileSatisfiesManifest(locked, deps)
		if !ok {
			t.Fatalf("expected satisfied, got reasons: %v", reasons)
		}
	})

	t.Run("missing_dep", func(t *testing.T) {
		locked := map[string]bool{"github.com/org/a": true}
		deps := []ManifestDep{{Key: "github.com/org/a"}, {Key: "github.com/org/b"}}
		ok, reasons := LockfileSatisfiesManifest(locked, deps)
		if ok {
			t.Fatal("expected not satisfied")
		}
		if len(reasons) != 1 {
			t.Fatalf("expected 1 reason, got %d", len(reasons))
		}
		if !strings.Contains(reasons[0], "github.com/org/b") {
			t.Fatalf("expected reason to mention missing dep, got %q", reasons[0])
		}
	})

	t.Run("local_deps_skipped", func(t *testing.T) {
		locked := map[string]bool{"github.com/org/a": true}
		deps := []ManifestDep{{Key: "github.com/org/a"}, {Key: "./local/path", IsLocal: true}}
		ok, _ := LockfileSatisfiesManifest(locked, deps)
		if !ok {
			t.Fatal("local deps should be skipped in frozen check")
		}
	})

	t.Run("empty_manifest", func(t *testing.T) {
		ok, reasons := LockfileSatisfiesManifest(map[string]bool{}, []ManifestDep{})
		if !ok || len(reasons) != 0 {
			t.Fatal("empty manifest should always be satisfied")
		}
	})
}

// TestParityRenderPlanText verifies the ASCII plan renderer.
func TestParityRenderPlanText(t *testing.T) {
	t.Run("empty_plan_no_verbose", func(t *testing.T) {
		p := UpdatePlan{}
		out := RenderPlanText(p, false)
		if out != "" {
			t.Fatalf("expected empty string, got %q", out)
		}
	})

	t.Run("add_entry_rendered", func(t *testing.T) {
		p := UpdatePlan{Entries: []PlanEntry{
			{
				DepKey:      "github.com/org/a",
				Action:      ActionAdd,
				DisplayName: "github.com/org/a",
				NewResolvedRef:    "main",
				NewResolvedCommit: "abc1234",
			},
		}}
		out := RenderPlanText(p, false)
		if !strings.Contains(out, "[+]") {
			t.Fatalf("expected add symbol [+], got %q", out)
		}
		if !strings.Contains(out, "github.com/org/a") {
			t.Fatalf("expected dep name in output, got %q", out)
		}
		if !strings.Contains(out, "1 added") {
			t.Fatalf("expected summary '1 added', got %q", out)
		}
	})

	t.Run("update_entry_rendered", func(t *testing.T) {
		p := UpdatePlan{Entries: []PlanEntry{
			{
				DepKey:            "github.com/org/b",
				Action:            ActionUpdate,
				DisplayName:       "github.com/org/b",
				OldResolvedRef:    "v1.0",
				OldResolvedCommit: "aaaaaaa0000000",
				NewResolvedRef:    "v2.0",
				NewResolvedCommit: "bbbbbbb1111111",
			},
		}}
		out := RenderPlanText(p, false)
		if !strings.Contains(out, "[~]") {
			t.Fatalf("expected update symbol [~], got %q", out)
		}
		if !strings.Contains(out, "1 updated") {
			t.Fatalf("expected summary '1 updated', got %q", out)
		}
	})

	t.Run("unchanged_hidden_without_verbose", func(t *testing.T) {
		p := UpdatePlan{Entries: []PlanEntry{
			{DepKey: "x", Action: ActionUnchanged, DisplayName: "x"},
			{DepKey: "y", Action: ActionAdd, DisplayName: "y", NewResolvedRef: "main"},
		}}
		out := RenderPlanText(p, false)
		if strings.Contains(out, "[=]") {
			t.Fatal("unchanged entry should be hidden without verbose")
		}
	})

	t.Run("unchanged_shown_with_verbose", func(t *testing.T) {
		p := UpdatePlan{Entries: []PlanEntry{
			{DepKey: "x", Action: ActionUnchanged, DisplayName: "x"},
		}}
		out := RenderPlanText(p, true)
		if !strings.Contains(out, "[=]") {
			t.Fatalf("unchanged entry should appear with verbose, got %q", out)
		}
	})

	t.Run("files_preview_truncated", func(t *testing.T) {
		p := UpdatePlan{Entries: []PlanEntry{
			{
				DepKey:        "z",
				Action:        ActionUpdate,
				DisplayName:   "z",
				DeployedFiles: []string{"a.md", "b.md", "c.md", "d.md", "e.md"},
			},
		}}
		out := RenderPlanText(p, false)
		if !strings.Contains(out, "+2 more") {
			t.Fatalf("expected '+2 more' truncation, got %q", out)
		}
	})
}

// TestParityInstallContext verifies that InstallContext mirrors the Python
// dataclass fields and defaults from context.py.
func TestParityInstallContext(t *testing.T) {
	t.Run("required_fields", func(t *testing.T) {
		ctx := NewInstallContext("/proj", "/proj/.apm")
		if ctx.ProjectRoot != "/proj" {
			t.Fatalf("unexpected ProjectRoot: %q", ctx.ProjectRoot)
		}
		if ctx.ApmDir != "/proj/.apm" {
			t.Fatalf("unexpected ApmDir: %q", ctx.ApmDir)
		}
	})

	t.Run("parallel_downloads_default", func(t *testing.T) {
		ctx := NewInstallContext("/proj", "/proj/.apm")
		if ctx.ParallelDownloads != 4 {
			t.Fatalf("expected default 4, got %d", ctx.ParallelDownloads)
		}
	})

	t.Run("maps_initialized", func(t *testing.T) {
		ctx := NewInstallContext("/proj", "/proj/.apm")
		if ctx.CallbackDownloaded == nil {
			t.Fatal("CallbackDownloaded should be initialized")
		}
		if ctx.PackageDeployedFiles == nil {
			t.Fatal("PackageDeployedFiles should be initialized")
		}
		if ctx.PackageTypes == nil {
			t.Fatal("PackageTypes should be initialized")
		}
		if ctx.PackageHashes == nil {
			t.Fatal("PackageHashes should be initialized")
		}
	})

	t.Run("bool_defaults_false", func(t *testing.T) {
		ctx := NewInstallContext("/proj", "/proj/.apm")
		if ctx.DryRun {
			t.Fatal("DryRun should default to false")
		}
		if ctx.Force {
			t.Fatal("Force should default to false")
		}
		if ctx.AllowInsecure {
			t.Fatal("AllowInsecure should default to false")
		}
		if ctx.DirectDepFailed {
			t.Fatal("DirectDepFailed should default to false")
		}
	})

	t.Run("installed_counts_zero", func(t *testing.T) {
		ctx := NewInstallContext("/proj", "/proj/.apm")
		if ctx.InstalledCount != 0 {
			t.Fatalf("InstalledCount should be 0, got %d", ctx.InstalledCount)
		}
		if ctx.TotalSkillsIntegrated != 0 {
			t.Fatalf("TotalSkillsIntegrated should be 0, got %d", ctx.TotalSkillsIntegrated)
		}
	})
}

// TestParityInstallRequest verifies that InstallRequest mirrors the Python
// dataclass from request.py.
func TestParityInstallRequest(t *testing.T) {
	t.Run("required_field", func(t *testing.T) {
		req := NewInstallRequest("pkg")
		if req.ApmPackage != "pkg" {
			t.Fatalf("unexpected ApmPackage: %v", req.ApmPackage)
		}
	})

	t.Run("parallel_downloads_default", func(t *testing.T) {
		req := NewInstallRequest(nil)
		if req.ParallelDownloads != 4 {
			t.Fatalf("expected default 4, got %d", req.ParallelDownloads)
		}
	})

	t.Run("bool_defaults_false", func(t *testing.T) {
		req := NewInstallRequest(nil)
		if req.UpdateRefs {
			t.Fatal("UpdateRefs should default to false")
		}
		if req.Force {
			t.Fatal("Force should default to false")
		}
		if req.Frozen {
			t.Fatal("Frozen should default to false")
		}
		if req.NoPolicy {
			t.Fatal("NoPolicy should default to false")
		}
	})

	t.Run("optional_fields_nil", func(t *testing.T) {
		req := NewInstallRequest(nil)
		if req.OnlyPackages != nil {
			t.Fatal("OnlyPackages should be nil by default")
		}
		if req.AllowProtocolFallback != nil {
			t.Fatal("AllowProtocolFallback should be nil by default")
		}
		if req.PlanCallback != nil {
			t.Fatal("PlanCallback should be nil by default")
		}
	})

	t.Run("plan_callback_invocable", func(t *testing.T) {
		called := false
		req := NewInstallRequest(nil)
		req.PlanCallback = func(p *UpdatePlan) bool {
			called = true
			return true
		}
		plan := &UpdatePlan{}
		result := req.PlanCallback(plan)
		if !called {
			t.Fatal("PlanCallback was not called")
		}
		if !result {
			t.Fatal("PlanCallback should return true")
		}
	})
}
