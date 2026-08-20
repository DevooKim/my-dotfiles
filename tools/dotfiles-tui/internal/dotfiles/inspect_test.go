package dotfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func symlinkFixture(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(newname), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(oldname, newname); err != nil {
		t.Fatal(err)
	}
}

func zshFixture(t *testing.T) (string, string, Package) {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, filepath.Join(PackageRoot(repo, "zsh"), ".zshrc"), "source")
	writeFixtureFile(t, filepath.Join(PackageRoot(repo, "zsh"), ".config", "zsh", "alias.zsh"), "alias")
	return repo, home, Package{Name: "zsh"}
}

func TestInspectPackageNotInstalled(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	inspection, err := InspectPackage(repo, home, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Status != NotInstalled {
		t.Fatalf("status = %q, want %q", inspection.Status, NotInstalled)
	}
}

func TestInspectPackageInstalledWithLeafLinks(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	symlinkFixture(t, filepath.Join(PackageRoot(repo, "zsh"), ".zshrc"), filepath.Join(home, ".zshrc"))
	symlinkFixture(t, filepath.Join(PackageRoot(repo, "zsh"), ".config", "zsh", "alias.zsh"), filepath.Join(home, ".config", "zsh", "alias.zsh"))

	inspection, err := InspectPackage(repo, home, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Status != Installed {
		t.Fatalf("status = %q, want %q", inspection.Status, Installed)
	}
}

func TestInspectPackageIncomplete(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	symlinkFixture(t, filepath.Join(PackageRoot(repo, "zsh"), ".zshrc"), filepath.Join(home, ".zshrc"))

	inspection, err := InspectPackage(repo, home, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Status != Incomplete {
		t.Fatalf("status = %q, want %q", inspection.Status, Incomplete)
	}
}

func TestInspectPackageInstalledWithStowDirectoryLink(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	symlinkFixture(t, filepath.Join(PackageRoot(repo, "zsh"), ".zshrc"), filepath.Join(home, ".zshrc"))
	symlinkFixture(t, filepath.Join(PackageRoot(repo, "zsh"), ".config", "zsh"), filepath.Join(home, ".config", "zsh"))

	inspection, err := InspectPackage(repo, home, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Status != Installed {
		t.Fatalf("status = %q, want %q", inspection.Status, Installed)
	}
}

func TestInspectPackageConflictForRegularFile(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	writeFixtureFile(t, filepath.Join(home, ".zshrc"), "user owned")

	inspection, err := InspectPackage(repo, home, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Status != Conflict {
		t.Fatalf("status = %q, want %q", inspection.Status, Conflict)
	}
}

func TestInspectPackageConflictForBrokenLink(t *testing.T) {
	repo, home, pkg := zshFixture(t)
	symlinkFixture(t, filepath.Join(home, "missing"), filepath.Join(home, ".zshrc"))

	inspection, err := InspectPackage(repo, home, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Status != Conflict {
		t.Fatalf("status = %q, want %q", inspection.Status, Conflict)
	}
}

func TestCatalogIsFlatAndComplete(t *testing.T) {
	want := []string{"aerospace", "ghostty", "hammerspoon", "herdr", "rift", "starship", "wezterm", "zed", "zsh"}
	got := Catalog()
	if len(got) != len(want) {
		t.Fatalf("catalog length = %d, want %d", len(got), len(want))
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Fatalf("catalog[%d] = %q, want %q", i, got[i].Name, name)
		}
	}
}
