class Schmournal < Formula
  desc "Terminal-based work journal"
  homepage "https://github.com/SleepyPxnda/schmournal"
  url "https://github.com/SleepyPxnda/schmournal/archive/refs/tags/PLACEHOLDER_TAG.tar.gz"
  sha256 "PLACEHOLDER_SHA256"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "."
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/schmournal --version")
  end
end
