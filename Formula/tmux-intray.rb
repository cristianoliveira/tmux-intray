class TmuxIntray < Formula
  desc "A quiet inbox for things that happen while you're not looking"
  homepage "https://github.com/cristianoliveira/tmux-intray"
  url "https://github.com/cristianoliveira/tmux-intray/archive/refs/heads/main.tar.gz"
  version "0.1.0"
  sha256 "e9f1e7817cdff92a1e17eaf024a2cca17359e4b23f45115137c27643b78cbcbc"
  license "MIT"

  head "https://github.com/cristianoliveira/tmux-intray.git", branch: "main"

  depends_on "bash"

  def install
    # Install all files to libexec
    libexec.install Dir["*"]
    # Symlink the main binary to bin
    bin.install_symlink libexec/"bin/tmux-intray"
  end

  test do
    system "#{bin}/tmux-intray", "version"
  end
end