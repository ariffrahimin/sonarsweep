class Sonarsweep < Formula
  desc "SonarQube Issue Exporter - Fetch issues to CSV beautifully"
  homepage "https://github.com/ariffrahimin/sonarsweep"
  version "VERSION_PLACEHOLDER"

  on_macos do
    on_arm do
      url "https://github.com/ariffrahimin/sonarsweep/releases/download/vVERSION_PLACEHOLDER/sonarsweep-darwin-arm64.tar.gz"
      sha256 "sha256-darwin-arm64 = CHECKSUM_PLACEHOLDER"
    end
    on_intel do
      url "https://github.com/ariffrahimin/sonarsweep/releases/download/vVERSION_PLACEHOLDER/sonarsweep-darwin-amd64.tar.gz"
      sha256 "sha256-darwin-amd64 = CHECKSUM_PLACEHOLDER"
    end
  end

  on_linux do
    url "https://github.com/ariffrahimin/sonarsweep/releases/download/vVERSION_PLACEHOLDER/sonarsweep-linux-amd64.tar.gz"
    sha256 "sha256-linux-amd64 = CHECKSUM_PLACEHOLDER"
  end

  def install
    bin.install "sonarsweep"
  end

  test do
    system "#{bin}/sonarsweep", "--version"
  end
end