class TmuxIntray < Formula
  desc "A quiet inbox for things that happen while you're not looking"
  homepage "https://github.com/cristianoliveira/tmux-intray"
  url "https://github.com/cristianoliveira/tmux-intray/archive/refs/heads/main.tar.gz"
  version "0.1.0"
  sha256 "54afc2329913aed280fb6a3ca4a7da25b95a2822af7fc7113b48d3b5587c2066"
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