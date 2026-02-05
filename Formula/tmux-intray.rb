class TmuxIntray < Formula
  desc "A quiet inbox for things that happen while you're not looking"
  homepage "https://github.com/cristianoliveira/tmux-intray"
  url "https://github.com/cristianoliveira/tmux-intray/archive/refs/tags/v0.1.0.tar.gz"
  version "0.1.0"
  sha256 "0019dfc4b32d63c1392aa264aed2253c1e0c2fb09216f8e2cc269bbfb8bb49b5"
  license "MIT"

  head "https://github.com/cristianoliveira/tmux-intray.git", branch: "main"

  livecheck do
    url :stable
    strategy :github_latest
  end

  depends_on "go" => :build

  def install
    # Build the Go binary
    system "go", "build", "-o", "tmux-intray", "./cmd/tmux-intray"
    # Install binary to bin
    bin.install "tmux-intray"
  end

  test do
    system bin/"tmux-intray", "version"
  end
end
