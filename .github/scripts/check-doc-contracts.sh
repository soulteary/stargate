#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
base_sha=${1:-}
temp_dir=""

cleanup() {
  if [[ -n "$temp_dir" && -d "$temp_dir" ]]; then
    rm -rf "$temp_dir"
  fi
}
trap cleanup EXIT

check_markdown_structure() {
  local root=$1
  local failed=false

  while IFS= read -r -d '' file; do
    local fences
    fences=$(grep -c '^```' "$file" || true)
    if (( fences % 2 != 0 )); then
      echo "Unclosed fenced code block: ${file#"$root"/}" >&2
      failed=true
    fi
  done < <(find "$root" -type f -name '*.md' -not -path '*/.git/*' -print0)

  if ! perl -MFile::Find -MFile::Basename -e '
    my $root = shift;
    my $failed = 0;
    find({
      no_chdir => 1,
      wanted => sub {
        return unless /\.md$/;
        my $file = $File::Find::name;
        open my $fh, "<", $file or die "open $file: $!";
        local $/;
        my $text = <$fh>;
        while ($text =~ /\[[^\]]*\]\(([^)]+)\)/g) {
          my $target = $1;
          $target =~ s/^<|>$//g;
          $target =~ s/\s+["\x27].*$//;
          next if $target =~ m{^(?:#|https?://|mailto:|data:)};
          (my $path = $target) =~ s/#.*$//;
          my $resolved = dirname($file) . "/" . $path;
          next if -e $resolved;
          warn "Broken relative link in $file: $target\n";
          $failed = 1;
        }
      }
    }, $root);
    exit $failed;
  ' "$root"; then
    failed=true
  fi

  [[ "$failed" == false ]]
}

contract_violation_count() {
  local root=$1
  perl -MFile::Find -e '
    my $root = shift;
    my $count = 0;

    my $config_path = "$root/src/internal/config/config.go";
    open my $config_fh, "<", $config_path or die "open $config_path: $!";
    local $/;
    my $source = <$config_fh>;
    $source =~ /var envVariables = \[\]\*EnvVariable\{(.*?)\}\s*\n/s
      or die "cannot find runtime environment-variable registry\n";
    my @symbols = $1 =~ /&([A-Za-z0-9_]+)/g;
    my @environment;
    for my $symbol (@symbols) {
      $source =~ /\b\Q$symbol\E\s*=\s*EnvVariable\{(.*?)\}/s
        or die "cannot find declaration for $symbol\n";
      my $block = $1;
      $block =~ /Name:\s*"([A-Z0-9_]+)"/
        or die "cannot find environment name for $symbol\n";
      push @environment, $1;
    }
    push @environment, "LOG_LEVEL";
    my %runtime_environment = map { $_ => 1 } @environment;

    for my $file (glob "$root/docs/*/CONFIG.md") {
      open my $fh, "<", $file or die "open $file: $!";
      local $/;
      my $text = <$fh>;
      for my $name (@environment) {
        next if index($text, "`$name`") >= 0;
        warn "Missing $name in $file\n";
        $count++;
      }

      my %documented;
      while ($text =~ /^(?:\|\s*|#{3,6}\s*)`([A-Z][A-Z0-9_]*)`/mg) {
        $documented{$1} = 1;
      }
      for my $name (sort keys %documented) {
        next if $runtime_environment{$name};
        warn "Documented setting $name is not registered at runtime in $file\n";
        $count++;
      }
    }

    my $constants_path = "$root/src/cmd/stargate/constants.go";
    open my $constants_fh, "<", $constants_path or die "open $constants_path: $!";
    local $/;
    my $constants_source = <$constants_fh>;
    my %route_constants = $constants_source =~ /\b(Route[A-Za-z0-9_]+)\s*=\s*"([^"]+)"/g;

    my $server_path = "$root/src/cmd/stargate/server.go";
    open my $server_fh, "<", $server_path or die "open $server_path: $!";
    local $/;
    my $server_source = <$server_fh>;
    $server_source =~ /(func setupRoutes\b.*?\n})\n\n\/\/ findAssetsPath/s
      or die "cannot find setupRoutes in $server_path\n";
    my $route_source = $1;
    my %runtime_routes;
    while ($route_source =~ /\bapp\.(Get|Post|Put|Patch|Delete)\(\s*(Route[A-Za-z0-9_]+|"[^"]+")/g) {
      my ($method, $expression) = (uc($1), $2);
      my $path;
      if ($expression =~ /^"([^"]+)"$/) {
        $path = $1;
      } else {
        $path = $route_constants{$expression}
          // die "cannot resolve route constant $expression\n";
      }
      $runtime_routes{"$method $path"} = 1;
    }
    # logger-kit registers all three methods for this administrative endpoint.
    $runtime_routes{"GET /log/level"} = 1;
    $runtime_routes{"PUT /log/level"} = 1;
    $runtime_routes{"POST /log/level"} = 1;

    my %mention_only = map { $_ => 1 } (
      "GET /health", # deprecated compatibility alias, intentionally not a primary heading
      "PUT /log/level", "POST /log/level", # documented in the GET /log/level section
    );
    for my $file (glob "$root/docs/*/API.md") {
      open my $fh, "<", $file or die "open $file: $!";
      local $/;
      my $text = <$fh>;
      my %headings = map { $_ => 1 }
        ($text =~ /^###\s+`((?:GET|POST|PUT|PATCH|DELETE) \/[^`]*)`\s*$/mg);
      for my $route (sort keys %runtime_routes) {
        if ($mention_only{$route}) {
          next if index($text, "`$route`") >= 0;
          warn "Missing compatibility route contract $route in $file\n";
          $count++;
          next;
        }
        next if $headings{$route};
        warn "Missing route heading $route in $file\n";
        $count++;
      }

      my ($confirm_section) = $text =~ /^### `POST \/totp\/enroll\/confirm`\s*\n(.*?)(?=^### |\z)/ms;
      if (!defined($confirm_section) ||
          $confirm_section !~ /`enroll_id`/ ||
          $confirm_section !~ /`code`/ ||
          $confirm_section !~ /application\/x-www-form-urlencoded/) {
        warn "Incomplete TOTP confirmation form contract in $file\n";
        $count++;
      }

      my ($login_section) = $text =~ /^### `POST \/_login`\s*\n(.*?)(?=^### |\z)/ms;
      if (!defined($login_section) ||
          $login_section !~ /challenge_id=ch_xxx&verify_code=123456/) {
        warn "Missing executable Warden verification login example in $file\n";
        $count++;
      }

      my ($send_section) = $text =~ /^### `POST \/_send_verify_code`\s*\n(.*?)(?=^### |\z)/ms;
      if (!defined($send_section) ||
          $send_section !~ /`deliver_via`/ ||
          $send_section !~ /`401 Unauthorized`/ ||
          $send_section !~ /`503 Service Unavailable`/) {
        warn "Incomplete verification-send form or status contract in $file\n";
        $count++;
      }
    }

    my @patterns = (
      ["retired Warden OTP setting", qr/\bWARDEN_OTP_(?:ENABLED|SECRET_KEY)\b/],
      ["legacy Redis setting", qr/\b(?:REDIS_ENABLED|REDIS_ADDR|REDIS_PASSWORD)\b/],
      ["stale Fiber version", qr/Fiber v2\.52/],
      ["invalid bcrypt command", qr/go run -c [^\n]*golang\.org\/x\/crypto\/bcrypt/],
      ["overstated Herald Key ID requirement", qr/(?:also requires|还需要) `HERALD_HMAC_KEY_ID`/],
      ["unsupported readiness claim", qr/(?:Enterprise-Grade|Enterprise Authentication|Battle-tested)/i],
      ["verification endpoint incorrectly claims JSON input", qr/(?:or|oder|ou|o|または|또는|或)\s+JSON\s*\(`application\/json`\)/i],
      ["container health check omits port 8080", qr{http://localhost/healthz}],
      ["metrics incorrectly described as new in v1", qr/(?:No metrics endpoint|无指标端点|Added Prometheus metrics)/],
      ["stale Go 1.26 requirement", qr/\bGo(?:\s+(?:Version|版本))?\s*[:：]?\s*1\.26(?:\.\d+)?\+?\b/i],
    );

    find({
      no_chdir => 1,
      wanted => sub {
        return unless /\.md$/;
        my $file = $File::Find::name;
        open my $fh, "<", $file or die "open $file: $!";
        local $/;
        my $text = <$fh>;

        for my $entry (@patterns) {
          my ($label, $pattern) = @$entry;
          while ($text =~ /$pattern/g) {
            warn "$label in $file\n";
            $count++;
          }
        }

        if ($file =~ m{/ARCHITECTURE\.md$}) {
          while ($text =~ /alpine:3\.24[^\n]*curl/gi) {
            warn "runtime image incorrectly claims curl in $file\n";
            $count++;
          }
        }

        while ($text =~ /#### `AUTH_REFRESH_ENABLED`.*?\|\s*\*\*(?:Default|默认值)\*\*\s*\|\s*`false`/sg) {
          warn "incorrect auth-refresh default in $file\n";
          $count++;
        }
        while ($text =~ /^(?:export )?PASSWORDS=(?![\x27"])[^\n]*(?:\||\$)/mg) {
          warn "unquoted password shell assignment in $file\n";
          $count++;
        }
        while ($text =~ /^\s*-e (?:PASSWORDS|LOGIN_PAGE_TITLE|LOGIN_PAGE_FOOTER_TEXT)=(?![\x27"])[^\n]*(?:\||\$|\s)/mg) {
          warn "unquoted docker environment value in $file\n";
          $count++;
        }
        while ($text =~ /^\s*-\s+PASSWORDS=bcrypt:[^\n]*(?<!\$)\$(?!\$)/mg) {
          warn "unescaped Compose bcrypt value in $file\n";
          $count++;
        }
        while ($text =~ /((?:^.*\n){0,2})^\s*curl\s+-H\s+["\x27]Stargate-Password:/mg) {
          my $context = $1;
          next if $context =~ /PASSWORD_HEADER_AUTH_ENABLED=true/;
          warn "header-auth command omits server enablement prerequisite in $file\n";
          $count++;
        }
      }
    }, "$root/docs", glob("$root/README*.md"));

    print "$count\n";
  ' "$root"
}

check_markdown_structure "$repo_root"
current_violations=$(contract_violation_count "$repo_root" 2>/dev/null)

if [[ -n "$base_sha" ]]; then
  temp_dir=$(mktemp -d)
  git archive "$base_sha" -- 'README*.md' docs src/internal/config/config.go \
    src/cmd/stargate/constants.go src/cmd/stargate/server.go \
    | tar -x -C "$temp_dir"
  base_violations=$(contract_violation_count "$temp_dir" 2>/dev/null)
  echo "Documentation contract violations: base=$base_violations current=$current_violations"
  if (( current_violations > base_violations )); then
    echo "Documentation contract violations increased." >&2
    contract_violation_count "$repo_root" >/dev/null
    exit 1
  fi
elif (( current_violations > 0 )); then
  echo "Documentation contract violations: $current_violations" >&2
  contract_violation_count "$repo_root" >/dev/null
  exit 1
fi

echo "Documentation contract checks passed."
