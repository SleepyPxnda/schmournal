class Schmournal < Formula
  desc "Terminal-based work journal"
  homepage "https://github.com/SleepyPxnda/schmournal"
  url "https://github.com/SleepyPxnda/schmournal/archive/refs/tags/v1.2.tar.gz"
  sha256 "481248bc1779035002234d56b2e10e87a15bd2fedc80ffb15e5cef578f1e2e93"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "."
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/schmournal --version")
  end
end
