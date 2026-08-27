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

  local endpoints=(
    '/_login' '/_send_verify_code' '/totp/enroll'
    '/totp/enroll/confirm' '/totp/revoke' '/_logout'
    '/_session_exchange' '/_auth' '/_step_up' '/metrics'
    '/log/level' '/healthz' '/readyz' '/health'
  )
  local api endpoint
  for api in "$root"/docs/*/API.md; do
    for endpoint in "${endpoints[@]}"; do
      if ! grep -Fq "$endpoint" "$api"; then
        echo "Missing endpoint $endpoint in ${api#"$root"/}" >&2
        failed=true
      fi
    done
  done

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

    for my $file (glob "$root/docs/*/CONFIG.md") {
      open my $fh, "<", $file or die "open $file: $!";
      local $/;
      my $text = <$fh>;
      for my $name (@environment) {
        next if index($text, "`$name`") >= 0;
        warn "Missing $name in $file\n";
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
