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

    sub console_prompt_command {
      my ($line) = @_;
      if ($line =~ m{^[ \t]*(?:
          [#\$>%]|
          [A-Za-z0-9_.-]+\@[A-Za-z0-9_.-]+(?::[^#\$%\r\n]*)?[#\$%]|
          \[[^\]\r\n]+\][#\$%]
        )[ \t]+(.*)$}x) {
        return (1, $1);
      }
      return (0, "");
    }

    sub shell_line_continues {
      my ($line) = @_;
      return 0 unless $line =~ /(\\+)$/;
      return length($1) % 2;
    }

    sub inline_code_contexts {
      my ($text) = @_;
      my @contexts;
      while ($text =~ /(?<!`)(`+)(?!`)(.*?)(?<!`)\1(?!`)/gs) {
        my $code = $2;
        # CommonMark normalizes line endings inside code spans to spaces.
        $code =~ s/\r?\n/ /g;
        push @contexts, $code if $code =~ /\bhtpasswd\b/;
      }
      return @contexts;
    }

    sub standalone_command_context {
      my ($line) = @_;
      $line =~ s/^[ \t]*(?:[\$>%][ \t]+)?//;
      return unless $line =~ /\bhtpasswd\b/;

      # A direct invocation remains deliberately strict so a prose sentence
      # beginning with "htpasswd" is not interpreted as shell. Lines beginning
      # with known shell prefixes/delegators are executable contexts; their
      # actual command words are validated later by the shell parser.
      return $line if $line =~ m{^(?:\S*/)?htpasswd[ \t]+-};
      return $line if $line =~ /^[A-Za-z_][A-Za-z0-9_]*=/;
      return $line if $line =~ m{^(?:(?:\S*/)?(?:
          sudo|command|exec|builtin|nohup|env|time|timeout|gtimeout|nice|xargs|
          find|eval|stdbuf|bash|dash|ksh|mksh|pdksh|sh|zsh
        )|if|then|elif|else|while|until|do|coproc|!)(?:[ \t]|$)}x;
      return;
    }

    sub markdown_command_contexts {
      my ($text) = @_;
      my @contexts;
      my ($fence_char, $fence_length, $capture_fence, $console_fence,
          $console_prompt_seen, $console_command_continuing,
          $fence_content, $console_commands);
      my $inline_chunk = "";

      for my $line (split /\n/, $text, -1) {
        $line =~ s/\r$//;
        if (defined $fence_char) {
          if ($line =~ /^\s*(\Q$fence_char\E+)\s*$/ && length($1) >= $fence_length) {
            if ($capture_fence) {
              my $context = $console_fence && $console_prompt_seen
                ? $console_commands
                : $fence_content;
              push @contexts, $context;
            }
            undef $fence_char;
            undef $fence_length;
            undef $capture_fence;
            $fence_content = "";
            $console_commands = "";
            $console_command_continuing = 0;
            next;
          }
          if ($capture_fence) {
            my ($has_prompt, $command) = console_prompt_command($line);
            if ($console_fence && $has_prompt) {
              $console_prompt_seen = 1;
              $console_commands .= "$command\n";
              $console_command_continuing = shell_line_continues($command);
            } elsif ($console_fence && $console_command_continuing) {
              $console_commands .= "$line\n";
              $console_command_continuing = shell_line_continues($line);
            } else {
              # Keep plain console blocks usable as command-only blocks. Once
              # any prompt is present, this buffer is transcript output and
              # only explicitly prompted lines are inspected.
              $fence_content .= "$line\n";
            }
          }
          next;
        }

        if ($line =~ /^\s*(`{3,}|~{3,})[ \t]*([A-Za-z0-9_-]*)/) {
          push @contexts, inline_code_contexts($inline_chunk);
          $inline_chunk = "";
          my ($marker, $info) = ($1, $2);
          $fence_char = substr($marker, 0, 1);
          $fence_length = length($marker);
          $capture_fence = $info eq "" ||
            $info =~ /^(?:bash|sh|shell|zsh|console|terminal)$/i;
          $console_fence = $info =~ /^(?:console|terminal)$/i;
          $console_prompt_seen = 0;
          $console_command_continuing = 0;
          $fence_content = "";
          $console_commands = "";
          next;
        }

        # Inline spans may cross physical lines, but never a blank-line block
        # boundary or a fenced block. Keep each prose block isolated.
        if ($line =~ /^[ \t]*$/) {
          push @contexts, inline_code_contexts($inline_chunk);
          $inline_chunk = "";
        } else {
          $inline_chunk .= "$line\n";
        }

        # Also protect an unfenced command which is written as a complete line.
        my $standalone = standalone_command_context($line);
        push @contexts, $standalone if defined $standalone;
      }
      if (defined $fence_char && $capture_fence) {
        my $context = $console_fence && $console_prompt_seen
          ? $console_commands
          : $fence_content;
        push @contexts, $context;
      }
      push @contexts, inline_code_contexts($inline_chunk);
      return @contexts;
    }

    sub dollar_paren_body {
      my ($text, $start) = @_;
      my $body_start = $start + 2;
      my $offset = $body_start;
      my $depth = 1;
      my $quote = "";
      my $escaped = 0;
      my $comment = 0;

      while ($offset < length($text)) {
        my $char = substr($text, $offset, 1);
        if ($comment) {
          $comment = 0 if $char eq "\n" || $char eq "\r";
          $offset++;
          next;
        }
        if ($quote eq "\x27") {
          $quote = "" if $char eq "\x27";
          $offset++;
          next;
        }
        if ($escaped) {
          $escaped = 0;
          $offset++;
          next;
        }
        if ($char eq "\\") {
          $escaped = 1;
          $offset++;
          next;
        }

        my $previous = $offset > $body_start
          ? substr($text, $offset - 1, 1)
          : "";
        if ($quote eq "" && $char eq "#" &&
            ($offset == $body_start || $previous =~ /[ \t\r\n;|&()]/)) {
          $comment = 1;
          $offset++;
          next;
        }

        # Nested modern substitutions have their own quoting context. Skip
        # them atomically so their parentheses cannot close the outer body.
        if ($char eq "\x24" && substr($text, $offset + 1, 1) eq "(" &&
            substr($text, $offset + 2, 1) ne "(" &&
            $quote ne "\x27") {
          my (undef, $end, $closed) = dollar_paren_body($text, $offset);
          if ($closed) {
            $offset = $end;
            next;
          }
        }

        # Parentheses in legacy command substitutions belong to that nested
        # shell context, not to the surrounding modern substitution.
        if ($char eq "`" && $quote ne "\x27") {
          $offset++;
          my $backtick_escaped = 0;
          while ($offset < length($text)) {
            my $nested = substr($text, $offset, 1);
            if ($backtick_escaped) {
              $backtick_escaped = 0;
            } elsif ($nested eq "\\") {
              $backtick_escaped = 1;
            } elsif ($nested eq "`") {
              $offset++;
              last;
            }
            $offset++;
          }
          next;
        }

        if ($char eq "\x27" && $quote eq "") {
          $quote = "\x27";
          $offset++;
          next;
        }
        if ($char eq "\"") {
          $quote = $quote eq "\"" ? "" : "\"";
          $offset++;
          next;
        }
        if ($quote eq "") {
          if ($char eq "(") {
            $depth++;
          } elsif ($char eq ")") {
            $depth--;
            if ($depth == 0) {
              return (
                substr($text, $body_start, $offset - $body_start),
                $offset + 1,
                1,
              );
            }
          }
        }
        $offset++;
      }
      return ("", length($text), 0);
    }

    sub command_substitution_contexts {
      my ($text) = @_;
      my @contexts;
      my $source = $text;
      my $filtered = "";
      my $quote = "";
      my $escaped = 0;
      my $comment = 0;
      my $offset = 0;

      while ($offset < length($source)) {
        my $char = substr($source, $offset, 1);
        if ($comment) {
          $filtered .= $char;
          $comment = 0 if $char eq "\n" || $char eq "\r";
          $offset++;
          next;
        }
        if ($quote eq "\x27") {
          $filtered .= $char;
          $quote = "" if $char eq "\x27";
          $offset++;
          next;
        }
        if ($escaped) {
          $filtered .= $char;
          $escaped = 0;
          $offset++;
          next;
        }
        if ($char eq "\\") {
          $filtered .= $char;
          $escaped = 1;
          $offset++;
          next;
        }
        if ($char eq "\x27") {
          $filtered .= $char;
          $quote = "\x27" unless $quote eq "\"";
          $offset++;
          next;
        }
        if ($char eq "\"") {
          $filtered .= $char;
          $quote = $quote eq "\"" ? "" : "\"";
          $offset++;
          next;
        }

        my $previous = $offset > 0
          ? substr($source, $offset - 1, 1)
          : "";
        if ($quote eq "" && $char eq "#" &&
            ($offset == 0 || $previous =~ /[ \t\r\n;|&()]/)) {
          $filtered .= $char;
          $comment = 1;
          $offset++;
          next;
        }

        my $modern_substitution =
          $char eq "\x24" && substr($source, $offset + 1, 1) eq "(" &&
          substr($source, $offset + 2, 1) ne "(";
        my $arithmetic_expansion =
          $char eq "\x24" && substr($source, $offset + 1, 2) eq "((";
        my $process_substitution =
          $quote eq "" && ($char eq "<" || $char eq ">") &&
          substr($source, $offset + 1, 1) eq "(";
        if ($modern_substitution || $arithmetic_expansion ||
            $process_substitution) {
          my ($body, $end, $closed) =
            dollar_paren_body($source, $offset);
          if ($closed) {
            if ($arithmetic_expansion) {
              # Arithmetic text is data, but substitutions nested inside it
              # still execute and must enter the normal context queue.
              my (undef, @nested) =
                command_substitution_contexts($body);
              push @contexts, @nested;
            } else {
              push @contexts, $body if $body =~ /\S/;
            }
            $filtered .= "__shell_substitution__";
            $offset = $end;
            next;
          }
        }

        if ($char ne "`") {
          $filtered .= $char;
          $offset++;
          next;
        }

        $offset++;
        my $body = "";
        my $body_escaped = 0;
        my $closed = 0;
        while ($offset < length($source)) {
          my $nested = substr($source, $offset, 1);
          if ($body_escaped) {
            $body .= $nested;
            $body_escaped = 0;
          } elsif ($nested eq "\\") {
            $body .= $nested;
            $body_escaped = 1;
          } elsif ($nested eq "`") {
            $closed = 1;
            $offset++;
            last;
          } else {
            $body .= $nested;
          }
          $offset++;
        }
        if ($closed) {
          push @contexts, $body if $closed && $body =~ /\S/;
          $filtered .= "__shell_substitution__";
        } else {
          $filtered .= "`$body";
        }
      }
      return ($filtered, @contexts);
    }

    sub array_assignment_subscript_at {
      my ($line, $opening) = @_;
      my $depth = 1;
      my $quote = "";
      my $escaped = 0;

      for (my $offset = $opening + 1;
           $offset < length($line);
           $offset++) {
        my $char = substr($line, $offset, 1);
        if ($quote eq "\x27") {
          $quote = "" if $char eq "\x27";
          next;
        }
        if ($quote eq "\"") {
          if ($escaped) {
            $escaped = 0;
          } elsif ($char eq "\\") {
            $escaped = 1;
          } elsif ($char eq "\"") {
            $quote = "";
          }
          next;
        }
        if ($escaped) {
          $escaped = 0;
          next;
        }
        if ($char eq "\\") {
          $escaped = 1;
          next;
        }
        if ($char eq "\x27" || $char eq "\"") {
          $quote = $char;
          next;
        }
        $depth++ if $char eq "[";
        if ($char eq "]") {
          $depth--;
          return substr($line, $offset + 1) =~ /^\+?=/
            if $depth == 0;
        }
      }
      return 0;
    }

    sub ansi_c_escape_at {
      my ($text, $offset) = @_;
      my $remaining = substr($text, $offset);
      return (chr(hex($1)), 2 + length($1))
        if $remaining =~ /^\\x([0-9A-Fa-f]{1,2})/;
      return (chr(hex($1)), 2 + length($1))
        if $remaining =~ /^\\u([0-9A-Fa-f]{1,4})/;
      return (chr(hex($1)), 2 + length($1))
        if $remaining =~ /^\\U([0-9A-Fa-f]{1,8})/;
      return (chr(oct($1)), 1 + length($1))
        if $remaining =~ /^\\([0-7]{1,3})/;
      if ($remaining =~ /^\\c(.)/s) {
        return (chr(ord(uc($1)) & 0x1f), 3);
      }
      if ($remaining =~ /^\\([abefnrtv\\\x27"?])/) {
        my %escape = (
          a => "\a", b => "\b", e => "\e", f => "\f", n => "\n",
          r => "\r", t => "\t", v => "\x0b", "\\" => "\\",
          "\x27" => "\x27", "\"" => "\"", "?" => "?",
        );
        return ($escape{$1}, 2);
      }
      return ("\\", 1);
    }

    sub heredoc_specs_in_line {
      my ($line, $arithmetic, $parameter, $shell) = @_;
      my @specs;
      my $quote = $shell->{quote};
      my $escaped = $shell->{escaped};
      my $offset = 0;

      while ($offset < length($line)) {
        my $char = substr($line, $offset, 1);
        if ($parameter->{depth} > 0) {
          if ($parameter->{quote} eq "\x27") {
            $parameter->{quote} = "" if $char eq "\x27";
          } elsif ($parameter->{quote} eq "\"") {
            if ($parameter->{escaped}) {
              $parameter->{escaped} = 0;
            } elsif ($char eq "\\") {
              $parameter->{escaped} = 1;
            } elsif ($char eq "\"") {
              $parameter->{quote} = "";
            }
          } elsif ($parameter->{escaped}) {
            $parameter->{escaped} = 0;
          } elsif ($char eq "\\") {
            $parameter->{escaped} = 1;
          } elsif ($char eq "\x27" || $char eq "\"") {
            $parameter->{quote} = $char;
          } elsif ($char eq "\x24" &&
                   substr($line, $offset + 1, 1) eq "{") {
            $parameter->{depth}++;
            $offset++;
          } elsif ($char eq "}") {
            $parameter->{depth}--;
          }
          $offset++;
          next;
        }
        if ($arithmetic->{depth} > 0) {
          if ($arithmetic->{quote} eq "\x27") {
            $arithmetic->{quote} = "" if $char eq "\x27";
          } elsif ($arithmetic->{quote} eq "\"") {
            if ($arithmetic->{escaped}) {
              $arithmetic->{escaped} = 0;
            } elsif ($char eq "\\") {
              $arithmetic->{escaped} = 1;
            } elsif ($char eq "\"") {
              $arithmetic->{quote} = "";
            }
          } elsif ($arithmetic->{escaped}) {
            $arithmetic->{escaped} = 0;
          } elsif ($char eq "\\") {
            $arithmetic->{escaped} = 1;
          } elsif ($char eq "\x27" || $char eq "\"") {
            $arithmetic->{quote} = $char;
          } elsif ($arithmetic->{delimiter} eq ")" && $char eq "(") {
            $arithmetic->{depth}++;
          } elsif ($arithmetic->{delimiter} eq ")" && $char eq ")") {
            $arithmetic->{depth}--;
          } elsif ($arithmetic->{delimiter} eq "]" && $char eq "[") {
            $arithmetic->{depth}++;
          } elsif ($arithmetic->{delimiter} eq "]" && $char eq "]") {
            $arithmetic->{depth}--;
          }
          $offset++;
          next;
        }
        if ($quote eq "\x27") {
          $quote = "" if $char eq "\x27";
          $offset++;
          next;
        }
        if ($quote eq "\"") {
          if ($escaped) {
            $escaped = 0;
          } elsif ($char eq "\\") {
            $escaped = 1;
          } elsif ($char eq "\"") {
            $quote = "";
          }
          $offset++;
          next;
        }
        if ($escaped) {
          $escaped = 0;
          $offset++;
          next;
        }
        if ($char eq "\\") {
          $escaped = 1;
          $offset++;
          next;
        }
        if ($char eq "\x27" || $char eq "\"") {
          $quote = $char;
          $offset++;
          next;
        }

        my $previous = $offset > 0 ? substr($line, $offset - 1, 1) : "";
        last if $char eq "#" &&
          ($offset == 0 || $previous =~ /[ \t;|&()]/);

        if ($char eq "\x24" && substr($line, $offset + 1, 1) eq "{") {
          $parameter->{depth} = 1;
          $parameter->{quote} = "";
          $parameter->{escaped} = 0;
          $offset += 2;
          next;
        }

        # `<<` is an arithmetic left shift inside arithmetic expansions,
        # commands, and indexed-array subscripts, not a heredoc operator. Skip
        # the complete balanced region before looking for redirections so a
        # shift cannot consume later documentation as a fictitious body.
        my $before = substr($line, 0, $offset);
        my $array_subscript = $char eq "[" &&
          $before =~ /(?:^|[ \t;|&()])(?:[A-Za-z_][A-Za-z0-9_]*)$/ &&
          array_assignment_subscript_at($line, $offset);
        my $arithmetic_prefix =
          substr($line, $offset, 3) eq "\x24((" ? 3 :
          substr($line, $offset, 2) eq "((" ? 2 :
          substr($line, $offset, 2) eq "\x24[" ? 2 :
          $array_subscript ? 1 : 0;
        if ($arithmetic_prefix) {
          my $legacy_brackets =
            substr($line, $offset, 2) eq "\x24[" || $array_subscript;
          $arithmetic->{depth} = $legacy_brackets ? 1 : 2;
          $arithmetic->{delimiter} = $legacy_brackets ? "]" : ")";
          $arithmetic->{quote} = "";
          $arithmetic->{escaped} = 0;
          $offset += $arithmetic_prefix;
          next;
        }

        if ($char ne "<" || substr($line, $offset + 1, 1) ne "<" ||
            substr($line, $offset + 2, 1) eq "<") {
          $offset++;
          next;
        }

        my $cursor = $offset + 2;
        my $strip_tabs = substr($line, $cursor, 1) eq "-";
        $cursor++ if $strip_tabs;
        $cursor++ while substr($line, $cursor, 1) =~ /[ \t]/;

        my $delimiter = "";
        my $delimiter_quote = "";
        my $delimiter_quoted = 0;
        my $delimiter_escaped = 0;
        while ($cursor < length($line)) {
          my $part = substr($line, $cursor, 1);
          if ($delimiter_quote eq "ansi") {
            if ($part eq "\x27") {
              $delimiter_quote = "";
              $cursor++;
            } elsif ($part eq "\\") {
              my ($decoded, $consumed) =
                ansi_c_escape_at($line, $cursor);
              $delimiter .= $decoded;
              $cursor += $consumed;
            } else {
              $delimiter .= $part;
              $cursor++;
            }
            next;
          }
          if ($delimiter_quote eq "\x27") {
            if ($part eq "\x27") {
              $delimiter_quote = "";
            } else {
              $delimiter .= $part;
            }
            $cursor++;
            next;
          }
          if ($delimiter_quote eq "\"") {
            if ($delimiter_escaped) {
              $delimiter .= $part;
              $delimiter_escaped = 0;
            } elsif ($part eq "\\") {
              $delimiter_escaped = 1;
            } elsif ($part eq "\"") {
              $delimiter_quote = "";
            } else {
              $delimiter .= $part;
            }
            $cursor++;
            next;
          }
          if ($part eq "\\") {
            $delimiter_quoted = 1;
            $cursor++;
            if ($cursor < length($line)) {
              $delimiter .= substr($line, $cursor, 1);
              $cursor++;
            }
            next;
          }
          if ($part eq "\x24" &&
              substr($line, $cursor + 1, 1) eq "\x27") {
            $delimiter_quoted = 1;
            $delimiter_quote = "ansi";
            $cursor += 2;
            next;
          }
          if ($part eq "\x27" || $part eq "\"") {
            $delimiter_quoted = 1;
            $delimiter_quote = $part;
            $cursor++;
            next;
          }
          last if $part =~ /[ \t\r\n;|&()<>]/;
          $delimiter .= $part;
          $cursor++;
        }

        if (length($delimiter)) {
          push @specs, {
            delimiter => $delimiter,
            literal => $delimiter_quoted,
            strip_tabs => $strip_tabs,
            body => "",
          };
        }
        $offset = $cursor > $offset ? $cursor : $offset + 1;
      }
      $shell->{quote} = $quote;
      # A final backslash consumes the physical line ending; it does not
      # escape the first character of the next line.
      $shell->{escaped} = 0;
      $arithmetic->{escaped} = 0;
      $parameter->{escaped} = 0;
      return @specs;
    }

    sub heredoc_filtered_contexts {
      my ($text) = @_;
      my @command_lines;
      my @expanding_bodies;
      my @pending;
      my %arithmetic =
        (depth => 0, delimiter => "", quote => "", escaped => 0);
      my %parameter = (depth => 0, quote => "", escaped => 0);
      my %shell = (quote => "", escaped => 0);

      for my $line (split /\n/, $text, -1) {
        $line =~ s/\r$//;
        if (@pending) {
          my $candidate = $line;
          $candidate =~ s/^\t+// if $pending[0]->{strip_tabs};
          if ($candidate eq $pending[0]->{delimiter}) {
            my $completed = shift @pending;
            push @expanding_bodies, $completed->{body}
              if !$completed->{literal} && $completed->{body} =~ /\S/;
          } elsif (!$pending[0]->{literal}) {
            $pending[0]->{body} .= "$line\n";
          }
          next;
        }

        push @command_lines, $line;
        push @pending,
          heredoc_specs_in_line(
            $line, \%arithmetic, \%parameter, \%shell);
      }
      for my $unfinished (@pending) {
        push @expanding_bodies, $unfinished->{body}
          if !$unfinished->{literal} && $unfinished->{body} =~ /\S/;
      }
      return (join("\n", @command_lines), @expanding_bodies);
    }

    sub shell_tokens {
      my ($text) = @_;
      my @tokens;
      my $token = "";
      my $in_token = 0;
      my $quote = "";
      my $escaped = 0;
      my $ansi_open = 0;
      my $ansi_skip = 0;

      for my $offset (0 .. length($text) - 1) {
        my $char = substr($text, $offset, 1);
        if ($ansi_skip > 0) {
          $ansi_skip--;
          next;
        }
        if ($quote eq "ansi") {
          if ($ansi_open) {
            $ansi_open = 0;
          } elsif ($char eq "\x27") {
            $quote = "";
          } elsif ($char eq "\\") {
            my ($decoded, $consumed) =
              ansi_c_escape_at($text, $offset);
            $token .= $decoded;
            $ansi_skip = $consumed - 1;
          } else {
            $token .= $char;
          }
          next;
        }
        if ($quote eq "\x27") {
          if ($char eq "\x27") {
            $quote = "";
          } else {
            $token .= $char;
          }
          next;
        }
        if ($quote eq "\"") {
          if ($escaped) {
            $token .= $char;
            $escaped = 0;
          } elsif ($char eq "\\") {
            my $next = $offset + 1 < length($text)
              ? substr($text, $offset + 1, 1)
              : "";
            if ($next =~ /^[\$`"\\]$/) {
              $escaped = 1;
            } else {
              $token .= $char;
            }
          } elsif ($char eq "\"") {
            $quote = "";
          } else {
            $token .= $char;
          }
          next;
        }
        if ($escaped) {
          $token .= $char;
          $escaped = 0;
          next;
        }
        if ($char eq "\\") {
          $escaped = 1;
          $in_token = 1;
          next;
        }
        if ($char eq "\x24" &&
            substr($text, $offset + 1, 1) eq "\x27") {
          $quote = "ansi";
          $ansi_open = 1;
          $in_token = 1;
          next;
        }
        if ($char eq "\x27" || $char eq "\"") {
          $quote = $char;
          $in_token = 1;
          next;
        }
        if ($char =~ /[ \t]/) {
          if ($in_token) {
            push @tokens, $token;
            $token = "";
            $in_token = 0;
          }
          next;
        }
        $token .= $char;
        $in_token = 1;
      }
      push @tokens, $token if $in_token;
      return @tokens;
    }

    sub flush_shell_segment {
      my ($segments, $segment_ref) = @_;
      push @$segments, $$segment_ref if $$segment_ref =~ /\S/;
      $$segment_ref = "";
    }

    sub shell_command_segments {
      my ($text) = @_;
      my @segments;
      my $segment = "";
      my $quote = "";
      my $escaped = 0;
      my $comment = 0;
      my $array_depth = 0;
      my $offset = 0;

      # A backslash-newline is part of the same logical shell command.
      $text =~ s/\\\r?\n[ \t]*/ /g;
      while ($offset < length($text)) {
        my $char = substr($text, $offset, 1);
        if ($comment) {
          if ($char eq "\n" || $char eq "\r") {
            flush_shell_segment(\@segments, \$segment);
            $comment = 0;
          }
          $offset++;
          next;
        }
        if ($quote eq "\x27") {
          $segment .= $char unless $array_depth;
          $quote = "" if $char eq "\x27";
          $offset++;
          next;
        }
        if ($quote eq "\"" || $quote eq "`") {
          if ($escaped) {
            $segment .= $char unless $array_depth;
            $escaped = 0;
            $offset++;
            next;
          }
          if ($char eq "\\") {
            my $next = $offset + 1 < length($text)
              ? substr($text, $offset + 1, 1)
              : "";
            $segment .= $char unless $array_depth;
            $escaped = 1 if $quote eq "`" || $next =~ /^[\$`"\\]$/;
            $offset++;
            next;
          }
          $segment .= $char unless $array_depth;
          $quote = "" if $char eq $quote;
          $offset++;
          next;
        }
        if ($escaped) {
          $segment .= $char unless $array_depth;
          $escaped = 0;
          $offset++;
          next;
        }
        if ($char eq "\\") {
          $segment .= $char unless $array_depth;
          $escaped = 1;
          $offset++;
          next;
        }
        if ($char eq "\x27" || $char eq "\"" || $char eq "`") {
          $quote = $char;
          $segment .= $char unless $array_depth;
          $offset++;
          next;
        }

        if ($array_depth) {
          $array_depth++ if $char eq "(";
          $array_depth-- if $char eq ")";
          $offset++;
          next;
        }

        # Parentheses immediately following an assignment word introduce a
        # Bash indexed/associative array literal. Its elements are data, not
        # commands. Command substitutions inside them have already been
        # extracted independently by command_substitution_contexts().
        if ($char eq "(" && $segment =~
            /(?:^|[ \t])(?:[A-Za-z_][A-Za-z0-9_]*)(?:\[[^\]\r\n]*\])?\+?=$/) {
          $segment .= "()";
          $array_depth = 1;
          $offset++;
          next;
        }

        my $previous = length($segment) ? substr($segment, -1, 1) : "";
        if ($char eq "#" && ($segment eq "" || $previous =~ /[ \t]/)) {
          $comment = 1;
          $offset++;
          next;
        }
        if ($char =~ /[\r\n;()]/) {
          flush_shell_segment(\@segments, \$segment);
          $offset++;
          next;
        }
        if (($char eq "{" || $char eq "}") &&
            ($previous eq "" || $previous =~ /[ \t\r\n;]/)) {
          my $next = $offset + 1 < length($text)
            ? substr($text, $offset + 1, 1)
            : "";
          if ($next eq "" || $next =~ /[ \t\r\n;]/) {
            flush_shell_segment(\@segments, \$segment);
            $offset++;
            next;
          }
        }
        if ($char eq "&") {
          my $next = $offset + 1 < length($text)
            ? substr($text, $offset + 1, 1)
            : "";
          if ($previous eq ">" || $previous eq "<" || $next eq ">") {
            $segment .= $char;
          } else {
            flush_shell_segment(\@segments, \$segment);
          }
          $offset++;
          next;
        }
        if ($char eq "|") {
          if ($previous eq ">") {
            $segment .= $char;
          } else {
            flush_shell_segment(\@segments, \$segment);
          }
          $offset++;
          next;
        }
        $segment .= $char;
        $offset++;
      }
      flush_shell_segment(\@segments, \$segment);
      return @segments;
    }

    sub redirection_details {
      my ($token) = @_;
      if ($token =~ /^((?:\d*(?:<<<|<<-|<<|<>|>>|<&|>&|>\||>|<)|&>>?))(.*)$/) {
        return (1, length($2) > 0);
      }
      return (0, 0);
    }

    sub wrapper_option_needs_operand {
      my ($wrapper, $option) = @_;
      return 0 if $option =~ /^--[^=]+=/;

      if ($wrapper eq "sudo") {
        return 1 if $option =~ /^--(?:chdir|close-from|group|host|other-user|prompt|role|type|user)$/;
        return 1 if $option =~ /^-[^-]*[CDghpRTuU]$/;
      } elsif ($wrapper eq "env") {
        return 1 if $option =~ /^--(?:chdir|split-string|unset)$/;
        return 1 if $option =~ /^-[^-]*[CSu]$/;
      } elsif ($wrapper eq "exec") {
        return 1 if $option eq "-a";
      } elsif ($wrapper eq "stdbuf") {
        return 1 if $option =~ /^--(?:input|output|error)$/;
        return 1 if $option =~ /^-[ioe]$/;
      }
      return 0;
    }

    sub command_basename {
      my ($command) = @_;
      $command =~ s{.*/}{};
      return $command;
    }

    sub command_word_index {
      my (@arguments) = @_;
      my %prefix = map { $_ => 1 }
        qw(if then elif else while until do coproc !);
      my %wrapper = map { $_ => 1 }
        qw(command exec builtin nohup env sudo time stdbuf);
      my $index = 0;

      while ($index < @arguments) {
        my $argument = $arguments[$index];
        if ($argument eq "\$" || $prefix{$argument}) {
          $index++;
          next;
        }
        if ($argument =~ /^[A-Za-z_][A-Za-z0-9_]*=/) {
          $index++;
          next;
        }

        my ($redirection, $has_target) = redirection_details($argument);
        if ($redirection) {
          $index += $has_target ? 1 : 2;
          next;
        }

        my $command_name = command_basename($argument);
        if ($wrapper{$command_name}) {
          my $wrapper = $command_name;
          $index++;
          while ($index < @arguments) {
            my $option = $arguments[$index];
            if ($option eq "--") {
              $index++;
              last;
            }
            last unless $option =~ /^-/ && $option ne "-";
            my $needs_operand =
              wrapper_option_needs_operand($wrapper, $option);
            $index++;
            $index++ if $needs_operand && $index < @arguments;
          }
          next;
        }
        return $index;
      }
      return -1;
    }

    sub htpasswd_command_index {
      my (@arguments) = @_;
      my $index = command_word_index(@arguments);
      return -1 if $index < 0;
      return $arguments[$index] =~ m{(?:^|/)htpasswd$} ? $index : -1;
    }

    sub shell_c_command {
      my (@arguments) = @_;
      my $command_index = command_word_index(@arguments);
      return unless $command_index >= 0;
      return unless $arguments[$command_index] =~
        m{(?:^|/)(?:ba|da|k|mk|pdk|z)?sh$};

      for (my $index = $command_index + 1;
           $index < @arguments;
           $index++) {
        my $argument = $arguments[$index];
        last if $argument eq "--";
        if ($argument =~ /^-[^-]*c/) {
          return $index + 1 < @arguments
            ? $arguments[$index + 1]
            : "";
        }
        if ($argument =~ /^(?:[-+]O|-o|--init-file|--rcfile)$/) {
          $index++;
          next;
        }
        last unless $argument =~ /^[-+]/;
      }
      return;
    }

    sub xargs_option_needs_operand {
      my ($option) = @_;
      return 0 if $option =~ /^--[^=]+=/;
      return 1 if $option =~ /^--(?:arg-file|delimiter|max-lines|max-args|max-procs|max-chars|process-slot-var)$/;
      return 1 if $option =~ /^-[^-]*[adEILnPs]$/;
      return 0;
    }

    sub xargs_command_arguments {
      my (@arguments) = @_;
      my $command_index = command_word_index(@arguments);
      return unless $command_index >= 0;
      return unless $arguments[$command_index] =~ m{(?:^|/)xargs$};

      my $index = $command_index + 1;
      while ($index < @arguments) {
        my $option = $arguments[$index];
        if ($option eq "--") {
          $index++;
          last;
        }
        last unless $option =~ /^-/ && $option ne "-";
        my $needs_operand = xargs_option_needs_operand($option);
        $index++;
        $index++ if $needs_operand && $index < @arguments;
      }
      return $index < @arguments ? @arguments[$index .. $#arguments] : ();
    }

    sub wrapper_index {
      my ($target, @arguments) = @_;
      my %prefix = map { $_ => 1 }
        qw(if then elif else while until do coproc !);
      my %wrapper = map { $_ => 1 }
        qw(command exec builtin nohup env sudo time stdbuf);
      my $index = 0;

      while ($index < @arguments) {
        my $argument = $arguments[$index];
        if ($argument eq "\x24" || $prefix{$argument} ||
            $argument =~ /^[A-Za-z_][A-Za-z0-9_]*=/) {
          $index++;
          next;
        }

        my ($redirection, $has_target) = redirection_details($argument);
        if ($redirection) {
          $index += $has_target ? 1 : 2;
          next;
        }

        my $command_name = command_basename($argument);
        return -1 unless $wrapper{$command_name};
        my $wrapper = $command_name;
        my $wrapper_index = $index;
        $index++;
        while ($index < @arguments) {
          my $option = $arguments[$index];
          if ($option eq "--") {
            $index++;
            last;
          }
          last unless $option =~ /^-/ && $option ne "-";
          my $needs_operand = wrapper_option_needs_operand($wrapper, $option);
          $index++;
          $index++ if $needs_operand && $index < @arguments;
        }
        return $wrapper_index if $wrapper eq $target;
      }
      return -1;
    }

    sub env_split_string {
      my (@arguments) = @_;
      my $env_index = wrapper_index("env", @arguments);
      return unless $env_index >= 0;

      for (my $index = $env_index + 1; $index < @arguments; $index++) {
        my $option = $arguments[$index];
        last if $option eq "--";
        return $arguments[$index + 1] // "" if
          $option eq "-S" || $option eq "--split-string" ||
          $option =~ /^-[^-]*S$/;
        return $1 if $option =~ /^--split-string=(.*)$/s;
        return $1 if $option =~ /^-[^-]*S(.+)$/s;
        last unless $option =~ /^-/ && $option ne "-";
        my $needs_operand = wrapper_option_needs_operand("env", $option);
        $index++ if $needs_operand && $index + 1 < @arguments;
      }
      return;
    }

    sub timeout_option_needs_operand {
      my ($option) = @_;
      return 0 if $option =~ /^--[^=]+=/;
      return 1 if $option =~ /^--(?:kill-after|signal)$/;
      return 1 if $option =~ /^-[^-]*[ks]$/;
      return 0;
    }

    sub timeout_command_arguments {
      my (@arguments) = @_;
      my $command_index = command_word_index(@arguments);
      return unless $command_index >= 0;
      return unless $arguments[$command_index] =~ m{(?:^|/)g?timeout$};

      my $index = $command_index + 1;
      while ($index < @arguments) {
        my $option = $arguments[$index];
        if ($option eq "--") {
          $index++;
          last;
        }
        last unless $option =~ /^-/ && $option ne "-";
        my $needs_operand = timeout_option_needs_operand($option);
        $index++;
        $index++ if $needs_operand && $index < @arguments;
      }

      # timeout requires one duration operand before the delegated command.
      $index++ if $index < @arguments;
      return $index < @arguments ? @arguments[$index .. $#arguments] : ();
    }

    sub nice_option_needs_operand {
      my ($option) = @_;
      return 0 if $option =~ /^--[^=]+=/;
      return 1 if $option eq "--adjustment";
      return 1 if $option =~ /^-[^-]*n$/;
      return 0;
    }

    sub nice_command_arguments {
      my (@arguments) = @_;
      my $command_index = command_word_index(@arguments);
      return unless $command_index >= 0;
      return unless $arguments[$command_index] =~ m{(?:^|/)nice$};

      my $index = $command_index + 1;
      while ($index < @arguments) {
        my $option = $arguments[$index];
        if ($option eq "--") {
          $index++;
          last;
        }
        last unless $option =~ /^-/ && $option ne "-";
        my $needs_operand = nice_option_needs_operand($option);
        $index++;
        $index++ if $needs_operand && $index < @arguments;
      }
      return $index < @arguments ? @arguments[$index .. $#arguments] : ();
    }

    sub find_exec_commands {
      my (@arguments) = @_;
      my $command_index = command_word_index(@arguments);
      return unless $command_index >= 0;
      return unless $arguments[$command_index] =~ m{(?:^|/)find$};

      my @commands;
      for (my $index = $command_index + 1;
           $index < @arguments;
           $index++) {
        next unless $arguments[$index] =~
          /^-(?:exec|execdir|ok|okdir)$/;
        my @command;
        $index++;
        while ($index < @arguments &&
               $arguments[$index] ne ";" &&
               $arguments[$index] ne "+") {
          push @command, $arguments[$index];
          $index++;
        }
        push @commands, \@command if @command;
      }
      return @commands;
    }

    sub eval_command {
      my (@arguments) = @_;
      my $command_index = command_word_index(@arguments);
      return unless $command_index >= 0;
      return unless $arguments[$command_index] eq "eval";
      my $first_argument = $command_index + 1;
      $first_argument++ if $first_argument < @arguments &&
        $arguments[$first_argument] eq "--";
      return join(" ", @arguments[$first_argument .. $#arguments])
        if $first_argument < @arguments;
      return "";
    }

    sub unsafe_htpasswd_count_in_arguments {
      my (@arguments) = @_;
      my $unsafe = 0;

      my $shell_command = shell_c_command(@arguments);
      $unsafe += unsafe_htpasswd_count_in_context($shell_command)
        if defined $shell_command && length($shell_command);

      my $evaluated_command = eval_command(@arguments);
      $unsafe += unsafe_htpasswd_count_in_context($evaluated_command)
        if defined $evaluated_command && length($evaluated_command);

      my $split_string = env_split_string(@arguments);
      if (defined $split_string && length($split_string)) {
        my @split_arguments = shell_tokens($split_string);
        $unsafe += unsafe_htpasswd_count_in_arguments(@split_arguments)
          if @split_arguments;
      }

      my @xargs_command = xargs_command_arguments(@arguments);
      $unsafe += unsafe_htpasswd_count_in_arguments(@xargs_command)
        if @xargs_command;

      my @timeout_command = timeout_command_arguments(@arguments);
      $unsafe += unsafe_htpasswd_count_in_arguments(@timeout_command)
        if @timeout_command;

      my @nice_command = nice_command_arguments(@arguments);
      $unsafe += unsafe_htpasswd_count_in_arguments(@nice_command)
        if @nice_command;

      for my $find_command (find_exec_commands(@arguments)) {
        $unsafe += unsafe_htpasswd_count_in_arguments(@$find_command);
      }

      my $command_index = htpasswd_command_index(@arguments);
      return $unsafe if $command_index < 0;

      for (my $index = $command_index + 1; $index < @arguments; $index++) {
        my $argument = $arguments[$index];
        last if $argument eq "--";
        my ($redirection, $has_target) = redirection_details($argument);
        if ($redirection) {
          $index++ unless $has_target;
          next;
        }
        if ($argument =~ /^-[A-Za-z]*b[A-Za-z]*$/) {
          $unsafe++;
          last;
        }
      }
      return $unsafe;
    }

    sub unsafe_htpasswd_count_in_context {
      my ($text) = @_;
      my $unsafe = 0;
      my @pending = ($text);

      # Heredoc recognition and command-substitution extraction must be
      # interleaved at every nesting level. A heredoc inside `$(...)` is not
      # visible while its outer quotes are still present, and its payload is
      # data except for substitutions in an expanding heredoc.
      while (@pending) {
        my $execution_context = shift @pending;
        my ($commands, @heredoc_bodies) =
          heredoc_filtered_contexts($execution_context);
        my ($filtered_commands, @substitutions) =
          command_substitution_contexts($commands);
        push @pending, @substitutions;
        for my $body (@heredoc_bodies) {
          my (undef, @body_substitutions) =
            command_substitution_contexts($body);
          push @pending, @body_substitutions;
        }

        for my $segment (shell_command_segments($filtered_commands)) {
          my @arguments = shell_tokens($segment);
          $unsafe += unsafe_htpasswd_count_in_arguments(@arguments);
        }
      }
      return $unsafe;
    }

    sub unsafe_htpasswd_command_count {
      my ($text) = @_;
      my $unsafe = 0;
      for my $context (markdown_command_contexts($text)) {
        $unsafe += unsafe_htpasswd_count_in_context($context);
      }
      return $unsafe;
    }

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
    );

    find({
      no_chdir => 1,
      wanted => sub {
        return unless /\.md$/;
        my $file = $File::Find::name;
        open my $fh, "<", $file or die "open $file: $!";
        local $/;
        my $text = <$fh>;

        # Inspect every complete logical htpasswd command. The batch-password
        # flag may follow options with their own arguments, but scanning must
        # stop before subsequent prose or shell commands.
        my $unsafe_htpasswd = unsafe_htpasswd_command_count($text);
        for (1 .. $unsafe_htpasswd) {
          warn "unsafe htpasswd batch-password option in $file\n";
          $count++;
        }

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
