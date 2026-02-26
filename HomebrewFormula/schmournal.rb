class Schmournal < Formula
  desc "Terminal-based work journal"
  homepage "https://github.com/SleepyPxnda/schmournal"
  url "https://github.com/SleepyPxnda/schmournal/archive/refs/tags/v1.3.tar.gz"
  sha256 "4e456b1b8de2a1f6dc835759b77f9365b28d8b4b8d36e80ca6bcf0e2b86e3a29"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "."
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/schmournal --version")
  end
end
