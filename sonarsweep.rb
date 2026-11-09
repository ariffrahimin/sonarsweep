class Sonarsweep < Formula
  desc "SonarQube Issue Exporter - Fetch issues to CSV beautifully"
  homepage "https://github.com/ariffrahimin/sonarsweep"
  version "VERSION_PLACEHOLDER"

  on_macos do
    on_arm do
      url "https://github.com/ariffrahimin/sonarsweep/releases/download/vVERSION_PLACEHOLDER/sonarsweep-darwin-arm64.tar.gz"
      sha256 "SHA256_DARWIN_ARM64_PLACEHOLDER"
    end
    on_intel do
      url "https://github.com/ariffrahimin/sonarsweep/releases/download/vVERSION_PLACEHOLDER/sonarsweep-darwin-amd64.tar.gz"
      sha256 "SHA256_DARWIN_AMD64_PLACEHOLDER"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ariffrahimin/sonarsweep/releases/download/vVERSION_PLACEHOLDER/sonarsweep-linux-arm64.tar.gz"
      sha256 "SHA256_LINUX_ARM64_PLACEHOLDER"
    end
    on_intel do
      url "https://github.com/ariffrahimin/sonarsweep/releases/download/vVERSION_PLACEHOLDER/sonarsweep-linux-amd64.tar.gz"
      sha256 "SHA256_LINUX_AMD64_PLACEHOLDER"
    end
  end

  def install
    bin.install "sonarsweep"
  end

  test do
    system "#{bin}/sonarsweep", "--version"
  end
end