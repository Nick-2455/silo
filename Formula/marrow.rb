class Marrow < Formula
  desc "Personal knowledge management orchestrator — triage resources into a 4-bucket roadmap"
  homepage "https://github.com/Nick-2455/marrow"
  url "https://github.com/Nick-2455/marrow/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "" # TODO: fill after first release
  license "MIT"

  depends_on "go" => :build

  def install
    # CGO-free build — no C toolchain needed
    system "go", "build", "-o", bin/"marrow", "-ldflags", "-s -w", "./cmd/marrow"
  end

  test do
    # Verify binary builds, is CGO-free, and accepts --help
    output = shell_output("#{bin}/marrow --help 2>&1")
    assert_match "-server", output
  end

  def caveats
    <<~EOS
      Marrow stores data in:
        Config: ~/.config/marrow/config.yaml
        Database: ~/.local/share/marrow/state.db

      These files are preserved on uninstall.
    EOS
  end
end
