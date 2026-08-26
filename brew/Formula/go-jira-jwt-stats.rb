class GoJiraJwtStats < Formula
  desc "Generate team Jira stats dashboards from browser-authenticated session cookies"
  homepage "https://github.com/dathan/go-jira-jwt-stats"
  url "https://github.com/dathan/go-jira-jwt-stats.git", using: :git #download strategy
  revision 1
  version 'master'
  head "https://github.com/dathan/go-jira-jwt-stats.git"

  depends_on "make" => :build
  depends_on "go" => :build

  def install
    ENV["GOPATH"] = buildpath
    path = buildpath/"src/github.com/dathan/go-jira-jwt-stats"
    path.install Dir["*"]
    cd path do
      system "make", "build"
      system "ls -ltarh"
    end

    bin.install path/"bin/go-jira-jwt-stats" => "go-jira-jwt-stats"
  end

  test do
    system "true"
  end
end
