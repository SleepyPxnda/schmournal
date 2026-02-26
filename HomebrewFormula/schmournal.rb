class Schmournal < Formula
  desc "Terminal-based work journal"
  homepage "https://github.com/SleepyPxnda/schmournal"
  url "https://github.com/SleepyPxnda/schmournal/archive/refs/tags/v1.4.tar.gz"
  sha256 "bc8459e91034f9583db705f61cf08b7382f13935d082e1f137f9e15e6109a830"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "."
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/schmournal --version")
  end
end
