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

  if ! perl -MFile::Find -MFile::Basename -e '
    my $root = shift;
    my $failed = 0;

    sub expand_tabs {
      my ($line) = @_;
      my $expanded = "";
      my $column = 0;
      for my $offset (0 .. length($line) - 1) {
        my $char = substr($line, $offset, 1);
        if ($char eq "\t") {
          my $width = 4 - ($column % 4);
          $expanded .= " " x $width;
          $column += $width;
        } else {
          $expanded .= $char;
          $column++;
        }
      }
      return $expanded;
    }

    sub consume_container {
      my ($line, $offset, $container) = @_;
      my $remaining = substr($line, $offset);

      if ($container->{type} eq "quote") {
        return -1 unless $remaining =~ /^( {0,3})>/;
        my $consumed = length($1) + 1;
        $consumed++ if substr($remaining, $consumed, 1) eq " ";
        return $offset + $consumed;
      }

      # Blank lines may remain in a list item without repeating its content
      # indentation. A non-blank continuation must provide the full width.
      return length($line) if $remaining =~ /^ *$/;
      my $width = $container->{width};
      return -1 if length($remaining) < $width;
      return -1 unless substr($remaining, 0, $width) eq " " x $width;
      return $offset + $width;
    }

    sub continue_containers {
      my ($line, $containers) = @_;
      my $offset = 0;
      my $matched = 0;
      for my $container (@$containers) {
        my $next = consume_container($line, $offset, $container);
        last if $next < 0;
        $offset = $next;
        $matched++;
      }
      return ($offset, $matched);
    }

    sub thematic_break {
      my ($line) = @_;
      return $line =~ /^ {0,3}(?:(?:\*[ ]*){3,}|(?:-[ ]*){3,}|(?:_[ ]*){3,})$/;
    }

    sub list_marker_details {
      my ($remaining) = @_;
      return unless
        $remaining =~ /^( {0,3})([-+*]|\d{1,9}[.)])(?:([ ]+)|$)/;
      my ($before, $marker, $spacing) = ($1, $2, $3);
      my $space_count = defined $spacing ? length($spacing) : 1;
      my $padding = $space_count <= 4 ? $space_count : 1;
      my $width = length($before) + length($marker) + $padding;
      my $content = length($remaining) >= $width
        ? substr($remaining, $width)
        : "";
      my $ordered = $marker =~ /^(\d{1,9})[.)]$/;
      my $start = $ordered ? 0 + $1 : 0;
      my $has_content = $content =~ /[^ ]/;
      return ($width, $ordered, $start, $has_content);
    }

    sub fence_opener_details {
      my ($content) = @_;
      return unless $content =~ /^ {0,3}(`{3,}|~{3,})(.*)$/;
      my ($marker, $info) = ($1, $2);
      my $char = substr($marker, 0, 1);
      # A backtick sequence whose info string contains a backtick is paragraph
      # text, not a fenced-code opener. Tilde info strings have no such rule.
      return if $char eq "`" && index($info, "`") >= 0;
      return ($char, length($marker));
    }

    sub setext_underline {
      my ($content) = @_;
      return $content =~ /^ {0,3}(?:=+|-+)[ \t]*$/;
    }

    sub reference_title {
      my ($content) = @_;
      # CommonMark backslash escapes may protect ASCII punctuation, including
      # the delimiter used by any of the three reference-title forms.
      return $content =~ m{^(?:
        "(?:\\[\x21-\x2f\x3a-\x40\x5b-\x60\x7b-\x7e]|[^"\\])*" |
        \x27(?:\\[\x21-\x2f\x3a-\x40\x5b-\x60\x7b-\x7e]|[^\x27\\])*\x27 |
        \((?:\\[\x21-\x2f\x3a-\x40\x5b-\x60\x7b-\x7e]|[^()\\])*\)
      )[ \t]*$}x;
    }

    sub valid_reference_label {
      my ($label) = @_;
      return 0 if length($label) > 999;
      return $label =~ /[^ \t]/;
    }

    sub link_reference_definition {
      my ($content) = @_;
      return unless $content =~ m{^[ ]{0,3}
        \[((?:\\[\x21-\x2f\x3a-\x40\x5b-\x60\x7b-\x7e]|[^\[\]\\])+)
        \]:[ \t]*
        (?:<[^<>\r\n]*>|[^ \t<>\r\n]+)(.*)$
      }x;
      my ($label, $remainder) = ($1, $2);
      return unless valid_reference_label($label);
      return (1, 0) if $remainder =~ /^[ \t]*$/;
      return unless $remainder =~ s/^[ \t]+//;
      return unless reference_title($remainder);
      return (1, 1);
    }

    sub reference_destination_details {
      my ($content) = @_;
      return unless $content =~ m{^[ ]{0,3}
        (?:<[^<>\r\n]*>|[^ \t<>\r\n]+)(.*)$
      }x;
      my $remainder = $1;
      return (1, 0) if $remainder =~ /^[ \t]*$/;
      return unless $remainder =~ s/^[ \t]+//;
      return unless reference_title($remainder);
      return (1, 1);
    }

    sub reference_title_line {
      my ($content) = @_;
      return 0 unless $content =~ /^ {0,3}(.*)$/;
      return reference_title($1);
    }

    sub following_reference_title_line {
      my ($containers, $line_index, $lines) = @_;
      return 0 if $line_index + 1 >= @$lines;

      my $title_line = $lines->[$line_index + 1];
      $title_line =~ s/\r$//;
      $title_line = expand_tabs($title_line);
      my ($offset, $matched) = continue_containers($title_line, $containers);
      return 0 if $matched < @$containers;
      return reference_title_line(substr($title_line, $offset));
    }

    sub multiline_reference_lines {
      my ($content, $containers, $line_index, $lines) = @_;
      return 0 unless $content =~ m{^[ ]{0,3}
        \[((?:\\[\x21-\x2f\x3a-\x40\x5b-\x60\x7b-\x7e]|[^\[\]\\])+)
        \]:[ \t]*$
      }x;
      return 0 unless valid_reference_label($1);
      return 0 if $line_index + 1 >= @$lines;

      my $destination_line = $lines->[$line_index + 1];
      $destination_line =~ s/\r$//;
      $destination_line = expand_tabs($destination_line);
      my ($offset, $matched) =
        continue_containers($destination_line, $containers);
      return 0 if $matched < @$containers;
      my @destination =
        reference_destination_details(substr($destination_line, $offset));
      return 0 unless @destination;

      my (undef, $has_title) = @destination;
      return 1 if $has_title || $line_index + 2 >= @$lines;

      my $title_line = $lines->[$line_index + 2];
      $title_line =~ s/\r$//;
      $title_line = expand_tabs($title_line);
      ($offset, $matched) = continue_containers($title_line, $containers);
      return 1 if $matched < @$containers;
      return reference_title_line(substr($title_line, $offset)) ? 2 : 1;
    }

    sub interrupts_paragraph {
      my ($remaining) = @_;
      return 1 if $remaining =~ /^ *$/;
      return 1 if $remaining =~ /^ {0,3}>/;
      my @fence = fence_opener_details($remaining);
      return 1 if @fence;
      return 1 if $remaining =~ /^ {0,3}#{1,6}(?:[ ]|$)/;
      return 1 if thematic_break($remaining);
      my ($starts_html, undef, undef, $html_interrupts) =
        html_block_start($remaining);
      return 1 if $starts_html && $html_interrupts;

      my @marker = list_marker_details($remaining);
      if (@marker) {
        my (undef, $ordered, $start, $has_content) = @marker;
        return 1 if $has_content && (!$ordered || $start == 1);
      }
      return 0;
    }

    sub paragraph_content {
      my ($remaining) = @_;
      return 0 if $remaining =~ /^ *$/;
      return 0 if $remaining =~ /^ {0,3}#{1,6}(?:[ ]|$)/;
      return 0 if thematic_break($remaining);
      return 1;
    }

    sub html_block_start {
      my ($content) = @_;
      return (1, qr/-->/, 0, 1) if $content =~ /^ {0,3}<!--/;
      return (1, qr/\?>/, 0, 1) if $content =~ /^ {0,3}<\?/;
      return (1, qr/\]\]>/, 0, 1) if $content =~ /^ {0,3}<!\[CDATA\[/;
      return (1, qr/>/, 0, 1) if $content =~ /^ {0,3}<![A-Za-z]/;

      if ($content =~ /^ {0,3}<(script|pre|style|textarea)(?:[ \t]|>|$)/i) {
        my $tag = $1;
        return (1, qr{</\Q$tag\E>}i, 0, 1);
      }

      if ($content =~ m{^[ ]{0,3}</?(?:
          address|article|aside|base|basefont|blockquote|body|caption|center|col|
          colgroup|dd|details|dialog|dir|div|dl|dt|fieldset|figcaption|figure|
          footer|form|frame|frameset|h[1-6]|head|header|hr|html|iframe|legend|
          li|link|main|menu|menuitem|nav|noframes|ol|optgroup|option|p|param|
          search|section|summary|table|tbody|td|tfoot|th|thead|title|tr|track|ul
        )(?:[ \t]|/?>|$)}ix) {
        return (1, undef, 1, 1);
      }

      # A complete open or closing tag for any other element starts a type-7
      # HTML block only where it does not interrupt an existing paragraph.
      if ($content =~ m{^ {0,3}</[A-Za-z][A-Za-z0-9-]*[ \t]*>[ \t]*$} ||
          $content =~ m{^ {0,3}<[A-Za-z][A-Za-z0-9-]*(?:[ \t]+[A-Za-z_:][A-Za-z0-9_.:-]*(?:[ \t]*=[ \t]*(?:[^\x00-\x20"\x27=<>`]+|"[^"]*"|\x27[^\x27]*\x27))?)*[ \t]*/?>[ \t]*$}) {
        return (1, undef, 1, 0);
      }
      return (0, undef, 0, 0);
    }

    sub html_block_finished {
      my ($content, $end_pattern, $until_blank) = @_;
      return $content =~ /^ *$/ if $until_blank;
      return $content =~ /$end_pattern/;
    }

    sub open_containers {
      my ($line, $offset, $containers, $paragraph_active) = @_;
      my $opened = 0;
      while ($offset <= length($line)) {
        my $remaining = substr($line, $offset);

        # Thematic breaks take precedence over interpreting their first marker
        # as a list item.
        last if thematic_break($remaining);

        if ($remaining =~ /^( {0,3})>/) {
          my $consumed = length($1) + 1;
          $consumed++ if substr($remaining, $consumed, 1) eq " ";
          push @$containers, { type => "quote" };
          $offset += $consumed;
          $paragraph_active = 0;
          $opened++;
          next;
        }

        my @marker = list_marker_details($remaining);
        if (@marker) {
          my ($width, $ordered, $start, $has_content) = @marker;
          # An ordered list can interrupt a paragraph only when it starts at
          # one, and an empty item cannot interrupt a paragraph at all.
          last if $paragraph_active &&
            (!$has_content || ($ordered && $start != 1));
          push @$containers, { type => "list", width => $width };
          $offset += $width;
          $offset = length($line) if $offset > length($line);
          $paragraph_active = 0;
          $opened++;
          next;
        }

        last;
      }
      return ($offset, $opened);
    }

    find({
      no_chdir => 1,
      wanted => sub {
        return unless /\.md$/;
        my $file = $File::Find::name;
        open my $fh, "<", $file or die "open $file: $!";
        local $/;
        my $text = <$fh>;

        my ($fence_char, $fence_length, $fence_line);
        my @containers;
        my @fence_containers;
        my $html_active = 0;
        my $html_end_pattern;
        my $html_until_blank = 0;
        my @html_containers;
        my $paragraph_active = 0;
        my $paragraph_depth = 0;
        my $line_number = 0;
        my $reference_lines_to_skip = 0;
        my @lines = split /\n/, $text, -1;
        LINE: for (my $line_index = 0;
                   $line_index < @lines;
                   $line_index++) {
          my $line = $lines[$line_index];
          $line_number = $line_index + 1;
          $line =~ s/\r$//;
          $line = expand_tabs($line);
          if ($reference_lines_to_skip > 0) {
            $reference_lines_to_skip--;
            next LINE;
          }

          # A container can end before its fenced block is explicitly closed.
          # Report that opening fence immediately, then parse the current line
          # again at the surviving outer container level.
          REPROCESS: while (1) {
            if (defined $fence_char) {
              my ($offset, $matched) =
                continue_containers($line, \@fence_containers);
              if ($matched < @fence_containers) {
                (my $relative = $file) =~ s{^\Q$root\E/?}{};
                warn "Unclosed fenced code block in $relative:$fence_line " .
                  "before container ended at line $line_number\n";
                $failed = 1;
                splice @containers, $matched;
                undef $fence_char;
                undef $fence_length;
                undef $fence_line;
                @fence_containers = ();
                $paragraph_active = 0;
                next REPROCESS;
              }

              my $candidate = substr($line, $offset);
              if ($candidate =~ /^ {0,3}(\Q$fence_char\E+)[ ]*$/ && length($1) >= $fence_length) {
                undef $fence_char;
                undef $fence_length;
                undef $fence_line;
                @fence_containers = ();
                $paragraph_active = 0;
              }
              next LINE;
            }

            if ($html_active) {
              my ($offset, $matched) =
                continue_containers($line, \@html_containers);
              if ($matched < @html_containers) {
                # Unlike a fenced block, a raw HTML block simply ends when
                # its list or block-quote container ends. Reprocess the line
                # at the surviving outer container level.
                splice @containers, $matched;
                $html_active = 0;
                undef $html_end_pattern;
                $html_until_blank = 0;
                @html_containers = ();
                $paragraph_active = 0;
                next REPROCESS;
              }

              my $html_content = substr($line, $offset);
              if (html_block_finished(
                  $html_content, $html_end_pattern, $html_until_blank)) {
                $html_active = 0;
                undef $html_end_pattern;
                $html_until_blank = 0;
                @html_containers = ();
              }
              next LINE;
            }

            my ($offset, $matched) = continue_containers($line, \@containers);
            if ($matched < @containers) {
              my $remaining = substr($line, $offset);
              if ($paragraph_active && $paragraph_depth == @containers &&
                  !interrupts_paragraph($remaining)) {
                # CommonMark permits paragraph text to lazily continue without
                # repeating one or more list container prefixes. Preserve the
                # stack so a following indented fence remains in that item.
                next LINE;
              }
              splice @containers, $matched;
              $paragraph_active = 0 if $paragraph_depth > $matched;
            }

            my $paragraph_here =
              $paragraph_active && $paragraph_depth == @containers;
            my $opened;
            ($offset, $opened) =
              open_containers($line, $offset, \@containers, $paragraph_here);
            if ($opened) {
              $paragraph_active = 0;
              $paragraph_here = 0;
            }
            my $content = substr($line, $offset);

            # Four spaces at the current container content column start an
            # indented code leaf when there is no paragraph to continue. It
            # must not create paragraph state that prevents a later non-one
            # ordered list from opening.
            if (!$paragraph_here && $content =~ /^ {4}/) {
              $paragraph_active = 0;
              next LINE;
            }

            if ($paragraph_here && setext_underline($content)) {
              $paragraph_active = 0;
              next LINE;
            }
            if (!$paragraph_here) {
              my @reference = link_reference_definition($content);
              if (@reference) {
                my (undef, $has_title) = @reference;
                if (!$has_title && following_reference_title_line(
                    \@containers, $line_index, \@lines)) {
                  $reference_lines_to_skip = 1;
                }
                $paragraph_active = 0;
                next LINE;
              }
            }
            if (!$paragraph_here) {
              my $continuation_lines = multiline_reference_lines(
                $content, \@containers, $line_index, \@lines);
              if ($continuation_lines > 0) {
                $reference_lines_to_skip = $continuation_lines;
                $paragraph_active = 0;
                next LINE;
              }
            }

            my ($starts_html, $html_end, $until_blank, $html_interrupts) =
              html_block_start($content);
            if ($starts_html && (!$paragraph_here || $html_interrupts)) {
              $paragraph_active = 0;
              unless (html_block_finished($content, $html_end, $until_blank)) {
                $html_active = 1;
                $html_end_pattern = $html_end;
                $html_until_blank = $until_blank;
                @html_containers = map { { %$_ } } @containers;
              }
              next LINE;
            }

            my ($opening_char, $opening_length) =
              fence_opener_details($content);
            if (defined $opening_char) {
              $fence_char = $opening_char;
              $fence_length = $opening_length;
              $fence_line = $line_number;
              @fence_containers = map { { %$_ } } @containers;
              $paragraph_active = 0;
              next LINE;
            }

            if (paragraph_content($content)) {
              $paragraph_active = 1;
              $paragraph_depth = scalar @containers;
            } else {
              $paragraph_active = 0;
            }
            next LINE;
          }
        }
        if (defined $fence_char) {
          (my $relative = $file) =~ s{^\Q$root\E/?}{};
          warn "Unclosed fenced code block in $relative:$fence_line\n";
          $failed = 1;
        }

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
      my ($confirm_request_section) = defined($confirm_section)
        ? ($confirm_section =~ /<!-- api-contract: totp-enroll-confirm-request-body -->\s*\n^#### [^\n]+\n(.*?)(?=^#### |\z)/ms)
        : undef;
      if (!defined($confirm_section) ||
          !defined($confirm_request_section) ||
          $confirm_section !~ /`enroll_id`/ ||
          $confirm_section !~ /`code`/ ||
          $confirm_request_section !~ /^\|\s*`application\/x-www-form-urlencoded`\s*\|\s*✅\s*\|/m ||
          $confirm_request_section !~ /^\|\s*`multipart\/form-data`\s*\|\s*✅\s*\|/m ||
          $confirm_request_section !~ /^\|\s*`application\/json`\s*\|\s*❌\s*\|/m ||
          $confirm_section !~ /`401 Unauthorized`/ ||
          $confirm_section !~ /10/) {
        warn "Incomplete TOTP confirmation form or authentication contract in $file\n";
        $count++;
      }

      my ($login_section) = $text =~ /^### `POST \/_login`\s*\n(.*?)(?=^### |\z)/ms;
      my %login_fields;
      if (defined($login_section)) {
        while ($login_section =~ /^\|\s*`([^`]+)`\s*\|/mg) {
          $login_fields{$1}++;
        }
      }
      my $missing_login_field = grep {
        ($login_fields{$_} // 0) < 1
      } qw(password phone mail challenge_id verify_code use_otp otp_code);
      if (!defined($login_section) ||
          ($login_fields{auth_method} // 0) < 2 ||
          ($login_fields{callback} // 0) < 2 ||
          $missing_login_field ||
          $login_section !~ /`phone` \+ `mail`/ ||
          $login_section !~ /`HERALD_TOTP_ENABLED=true`/ ||
          $login_section !~ /auth_method=password&password=yourpassword/ ||
          $login_section !~ /auth_method=warden&mail=user\@example\.com&challenge_id=ch_xxx&verify_code=123456&callback=app\.example\.com/ ||
          $login_section !~ /auth_method=warden&mail=user\@example\.com&use_otp=true&otp_code=123456&callback=app\.example\.com/ ||
          $login_section !~ /`400 Bad Request`/ ||
          $login_section !~ /`401 Unauthorized`/ ||
          $login_section !~ /`502 Bad Gateway`/ ||
          $login_section !~ /`503 Service Unavailable`/ ||
          $login_section !~ /`500 Internal Server Error`/) {
        warn "Incomplete password/Warden challenge/TOTP login contract in $file\n";
        $count++;
      }

      my ($send_section) = $text =~ /^### `POST \/_send_verify_code`\s*\n(.*?)(?=^### |\z)/ms;
      my ($send_request_section) = defined($send_section)
        ? ($send_section =~ /<!-- api-contract: send-verify-code-request-body -->\s*\n^#### [^\n]+\n(.*?)(?=^#### |\z)/ms)
        : undef;
      if (!defined($send_section) ||
          !defined($send_request_section) ||
          $send_section !~ /`deliver_via`/ ||
          $send_section !~ /`deliver_via=dingtalk`/ ||
          $send_section !~ /`phone` \+ `mail`/ ||
          $send_request_section !~ /^\|\s*`application\/x-www-form-urlencoded`\s*\|\s*✅\s*\|/m ||
          $send_request_section !~ /^\|\s*`multipart\/form-data`\s*\|\s*✅\s*\|/m ||
          $send_request_section !~ /^\|\s*`application\/json`\s*\|\s*❌\s*\|/m ||
          $send_section !~ /`401 Unauthorized`/ ||
          $send_section !~ /`503 Service Unavailable`/) {
        warn "Incomplete verification-send form or status contract in $file\n";
        $count++;
      }

      my ($enroll_page_section) = $text =~ /^### `GET \/totp\/enroll`\s*\n(.*?)(?=^### |\z)/ms;
      my ($enroll_start_section) = $text =~ /^### `POST \/totp\/enroll`\s*\n(.*?)(?=^### |\z)/ms;
      my ($revoke_page_section) = $text =~ /^### `GET \/totp\/revoke`\s*\n(.*?)(?=^### |\z)/ms;
      my ($revoke_confirm_section) = $text =~ /^### `POST \/totp\/revoke`\s*\n(.*?)(?=^### |\z)/ms;
      if (!defined($enroll_page_section) ||
          $enroll_page_section !~ /10/ ||
          $enroll_page_section !~ /`\/_login`/ ||
          $enroll_page_section !~ /`302 Found`/ ||
          $enroll_page_section !~ /`401 Unauthorized`/ ||
          !defined($enroll_start_section) ||
          $enroll_start_section !~ /10/ ||
          $enroll_start_section !~ /`\/_login`/ ||
          $enroll_start_section !~ /`302 Found`/ ||
          $enroll_start_section !~ /`401 Unauthorized`/ ||
          !defined($revoke_page_section) ||
          $revoke_page_section !~ /`\/_login`/ ||
          $revoke_page_section !~ /`302 Found`/ ||
          !defined($revoke_confirm_section) ||
          $revoke_confirm_section !~ /`password`/ ||
          $revoke_confirm_section !~ /`code`/ ||
          $revoke_confirm_section !~ /`401 Unauthorized`/) {
        warn "Incomplete TOTP session or reauthentication contract in $file\n";
        $count++;
      }
    }

    for my $file (glob "$root/docs/*/SECURITY.md") {
      open my $fh, "<", $file or die "open $file: $!";
      local $/;
      my $text = <$fh>;
      for my $term ("DEBUG=true", "DEBUG=false", "HERALD_TEST_MODE", "debug_code", "POST /_send_verify_code") {
        next if index($text, $term) >= 0;
        warn "Missing debug verification-code security contract $term in $file\n";
        $count++;
      }
    }

    for my $file (glob "$root/docs/*/DEPLOYMENT.md") {
      open my $fh, "<", $file or die "open $file: $!";
      local $/;
      my $text = <$fh>;
      for my $term (
        "SESSION_STORAGE_ENABLED=false",
        "SESSION_STORAGE_ENABLED=true",
        "SESSION_STORAGE_REDIS_*",
        "SESSION_EXCHANGE_SECRET",
      ) {
        next if index($text, $term) >= 0;
        warn "Missing session-storage deployment scope $term in $file\n";
        $count++;
      }
      for my $term (
        "stargate-v0.12.0.env",
        "stargate-v1.env",
        "WARDEN_OTP_ENABLED",
        "WARDEN_OTP_SECRET_KEY",
        "HERALD_ENABLED=true",
        "HERALD_TOTP_ENABLED=true",
        "HERALD_URL",
        "HERALD_TOTP_ENCRYPTION_KEY",
        "--env-file ./stargate-v1.env",
        q{old_container=${STARGATE_OLD_CONTAINER:-stargate}},
        q{mktemp "${old_env}.tmp.XXXXXX"},
        qq{docker inspect --format \x27{{range .Config.Env}}{{println .}}{{end}}\x27 "\$old_container"},
        q{ln "$export_tmp" "$old_env"},
        q{chmod 600 "$old_env"},
        q{set -C; cat "$old_env" > "$rollback_env"},
        q{WARDEN_OTP_(ENABLED|SECRET_KEY)},
        q{PORT[[:space:]]*(=|$)},
        q{print "PORT=8080"},
      ) {
        next if index($text, $term) >= 0;
        warn "Missing safe v1 environment migration contract $term in $file\n";
        $count++;
      }
      if ($text !~ /`WARDEN_ENABLED=true`[^\n]*`WARDEN_URL`/) {
        warn "Missing Warden prerequisites for Herald TOTP migration in $file\n";
        $count++;
      }
    }

    for my $file (glob "$root/docs/*/MIGRATION_V1.md") {
      open my $fh, "<", $file or die "open $file: $!";
      local $/;
      my $text = <$fh>;
      for my $term (
        "stargate-v0.12.0.env",
        "stargate-v1.env",
        "WARDEN_OTP_ENABLED",
        "WARDEN_OTP_SECRET_KEY",
        "HERALD_ENABLED=true",
        "HERALD_TOTP_ENABLED=true",
        "HERALD_URL",
        "HERALD_TOTP_ENCRYPTION_KEY",
        "--env-file ./stargate-v1.env",
        q{old_container=${STARGATE_OLD_CONTAINER:-stargate}},
        q{mktemp "${old_env}.tmp.XXXXXX"},
        qq{docker inspect --format \x27{{range .Config.Env}}{{println .}}{{end}}\x27 "\$old_container"},
        q{ln "$export_tmp" "$old_env"},
        q{chmod 600 "$old_env"},
        q{set -C; cat "$old_env" > "$rollback_env"},
        q{WARDEN_OTP_(ENABLED|SECRET_KEY)},
        q{PORT[[:space:]]*(=|$)},
        q{print "PORT=8080"},
      ) {
        next if index($text, $term) >= 0;
        warn "Missing v1 migration environment contract $term in $file\n";
        $count++;
      }
      if ($text !~ /`WARDEN_ENABLED=true`[^\n]*`WARDEN_URL`/) {
        warn "Missing Warden prerequisites for Herald TOTP migration in $file\n";
        $count++;
      }
    }

    my $changelog_path = "$root/CHANGELOG.md";
    open my $changelog_fh, "<", $changelog_path or die "open $changelog_path: $!";
    local $/;
    my $changelog = <$changelog_fh>;
    my ($v1_section) = $changelog =~ /^## \[1\.0\.0\][^\n]*\n(.*?)(?=^## |\z)/ms;
    my ($breaking_changes) = defined($v1_section)
      ? $v1_section =~ /^### Breaking changes\s*\n(.*?)(?=^### |\z)/ms
      : undef;
    if (!defined($breaking_changes)) {
      warn "Missing v1.0.0 Breaking changes section in $changelog_path\n";
      $count++;
    } else {
      if ($breaking_changes !~ /container[^\n]*`8080`[^\n]*`80`/i) {
        warn "Missing container port 80 to 8080 migration in v1.0.0 Breaking changes\n";
        $count++;
      }
      if (index($breaking_changes, "PASSWORD_HEADER_AUTH_ENABLED=true") < 0) {
        warn "Missing password-header authentication opt-in in v1.0.0 Breaking changes\n";
        $count++;
      }
    }

    my @patterns = (
      ["legacy Redis setting", qr/\b(?:REDIS_ENABLED|REDIS_ADDR|REDIS_PASSWORD)\b/],
      ["stale Fiber version", qr/Fiber v2\.52/],
      ["invalid bcrypt command", qr/go run -c [^\n]*golang\.org\/x\/crypto\/bcrypt/],
      ["overstated Herald Key ID requirement", qr/(?:also requires|还需要) `HERALD_HMAC_KEY_ID`/],
      ["unsupported readiness claim", qr/(?:Enterprise-Grade|Enterprise Authentication|Battle-tested)/i],
      ["verification endpoint incorrectly claims JSON input", qr/(?:or|oder|ou|o|または|또는|或)\s+JSON\s*\(`application\/json`\)/i],
      ["container health check omits port 8080", qr{http://localhost/healthz}],
      ["metrics incorrectly described as new in v1", qr/(?:No metrics endpoint|无指标端点|Added Prometheus metrics)/],
      ["stale Go 1.26 requirement", qr/\bGo(?:\s+(?:Version|版本))?\s*[:：]?\s*1\.26(?:\.\d+)?\+?\b/i],
      ["unsafe htpasswd batch-password option", qr/\bhtpasswd\s+-[A-Za-z]*b[A-Za-z]*\b/],
    );

    find({
      no_chdir => 1,
      wanted => sub {
        return unless /\.md$/;
        my $file = $File::Find::name;
        open my $fh, "<", $file or die "open $file: $!";
        local $/;
        my $text = <$fh>;

        my $retired_text = $text;
        if ($file =~ m{/(?:DEPLOYMENT|MIGRATION_V1)\.md$}) {
          # Migration guidance may name retired settings only as exact inline-code
          # identifiers. Any remaining occurrence is an executable or ambiguous
          # use and would make v1 reject the environment at startup.
          $retired_text =~ s/`WARDEN_OTP_ENABLED`//g;
          $retired_text =~ s/`WARDEN_OTP_SECRET_KEY`//g;
        }
        while ($retired_text =~ /\bWARDEN_OTP_(?:ENABLED|SECRET_KEY)\b/g) {
          warn "active or ambiguous retired Warden OTP setting in $file\n";
          $count++;
        }

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

    my $compose_path = "$root/docker-compose.yml";
    open my $compose_fh, "<", $compose_path or die "open $compose_path: $!";
    local $/;
    my $compose = <$compose_fh>;
    my ($compose_network_key, $compose_network_name) =
      $compose =~ /^networks:\s*\n\s{2}([^:\s]+):\s*\n\s{4}name:\s*([^\s#]+)/m;
    if (!defined($compose_network_key) || !defined($compose_network_name)) {
      warn "Cannot resolve the Compose-managed Traefik network in $compose_path\n";
      $count++;
    } elsif ($compose_network_key ne "traefik" ||
             $compose_network_name ne "stargate-traefik") {
      warn "Bundled Compose must preserve logical key traefik and actual name stargate-traefik in $compose_path\n";
      $count++;
    } else {
      my $attached_services = () = $compose =~ /^\s{6}-\s+\Q$compose_network_key\E\s*$/mg;
      if ($attached_services < 3) {
        warn "Not every bundled service uses logical network key $compose_network_key in $compose_path\n";
        $count++;
      }
      my @label_networks = $compose =~ /traefik\.docker\.network=([^\s"]+)/g;
      if (!@label_networks) {
        warn "Missing Traefik Docker network labels in $compose_path\n";
        $count++;
      }
      for my $label_network (@label_networks) {
        next if $label_network eq $compose_network_name;
        warn "Traefik label network $label_network does not match $compose_network_name in $compose_path\n";
        $count++;
      }
    }

    my ($compose_subnet) = $compose =~ /^\s*-\s+subnet:\s*([^\s#]+)/m;
    my ($trusted_proxies) = $compose =~ /^\s*-\s+TRUSTED_PROXIES=([^\s#]+)/m;
    if (!defined($compose_subnet) || !defined($trusted_proxies) ||
        $compose_subnet ne $trusted_proxies) {
      warn "Compose subnet and TRUSTED_PROXIES differ in $compose_path\n";
      $count++;
    }

    my @deployment_files = glob "$root/docs/*/DEPLOYMENT.md";
    if (!@deployment_files) {
      warn "No localized deployment guides found\n";
      $count++;
    }
    my $external_name = q!${TRAEFIK_NETWORK_NAME:-traefik}!;
    my $external_label = "traefik.docker.network=$external_name";
    for my $file (@deployment_files) {
      open my $fh, "<", $file or die "open $file: $!";
      local $/;
      my $text = <$fh>;
      my @requirements = (
        ["bundled network name", "stargate-traefik"],
        ["bundled Compose validation", "docker compose config"],
        ["external network name export", qq{export TRAEFIK_NETWORK_NAME="$external_name"}],
        ["existing network inspection", qq{docker network inspect "\$TRAEFIK_NETWORK_NAME"}],
        ["explicit external subnet creation", "--subnet 172.30.0.0/24"],
        ["created subnet export", "export TRAEFIK_NETWORK_CIDR=172.30.0.0/24"],
        ["inspected trusted-proxy CIDR", q!TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}!],
        ["external Compose network name", "name: $external_name"],
        ["external network declaration", "external: true"],
        ["external Traefik label", $external_label],
      );
      for my $requirement (@requirements) {
        my ($label, $needle) = @$requirement;
        next if index($text, $needle) >= 0;
        warn "Missing $label contract in $file\n";
        $count++;
      }
      if ($text =~ /TRUSTED_PROXIES=172\.30\.0\.0\/24/ ||
          $text =~ /traefik\.docker\.network=traefik/ ||
          $text =~ /docker network create traefik/) {
        warn "Hard-coded external Traefik network contract in $file\n";
        $count++;
      }
      while ($text =~ /traefik\.docker\.network=([^\s"]+)/g) {
        next if $1 eq $external_name;
        warn "External Traefik label does not use the configured network name in $file\n";
        $count++;
      }
    }
    print "$count\n";
  ' "$root"
}

check_markdown_structure "$repo_root"
current_violations=$(contract_violation_count "$repo_root" 2>/dev/null)

if [[ -n "$base_sha" ]]; then
  temp_dir=$(mktemp -d)
  git archive "$base_sha" -- 'README*.md' CHANGELOG.md docker-compose.yml docs src/internal/config/config.go \
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
