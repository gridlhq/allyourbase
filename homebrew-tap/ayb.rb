class Ayb < Formula
  desc "Backend-as-a-Service for PostgreSQL. Single binary, one config file."
  homepage "https://github.com/gridlhq/allyourbase"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/gridlhq/allyourbase/releases/download/v#{version}/ayb_#{version}_darwin_arm64.tar.gz"
      sha256 "6bc653e260485f472cddc945d3880f45f71470062282aab24e74a5f4bdc16c46"
    end
    on_intel do
      url "https://github.com/gridlhq/allyourbase/releases/download/v#{version}/ayb_#{version}_darwin_amd64.tar.gz"
      sha256 "ce21bb34e94f9fd09b748d7f476fe24c3f294a3669cf308dc63f71fa757937b9"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/gridlhq/allyourbase/releases/download/v#{version}/ayb_#{version}_linux_arm64.tar.gz"
      sha256 "e8e7034e0b63a2312f6126659f11355d8a98e9c9c964629d5abb00c27b869a92"
    end
    on_intel do
      url "https://github.com/gridlhq/allyourbase/releases/download/v#{version}/ayb_#{version}_linux_amd64.tar.gz"
      sha256 "3a055d1cdf9b6bdcc62c38895cb7c3718e48abfa56c68428ba3e1358844c43a6"
    end
  end

  def install
    bin.install "ayb"
  end

  test do
    assert_match "ayb", shell_output("#{bin}/ayb version")
  end
end
