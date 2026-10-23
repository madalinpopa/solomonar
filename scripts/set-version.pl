use strict;
use warnings;

my ($next) = @ARGV;
die "usage: perl scripts/set-version.pl <major.minor.patch>\n"
    unless @ARGV == 1 && $next =~ /\A\d+\.\d+\.\d+\z/;

open my $rules, '<', 'assets/framework/rules.md' or die "read rules: $!\n";
my ($current) = map { /^Framework version: (\d+\.\d+\.\d+)$/ ? $1 : () } <$rules>;
close $rules or die "close rules: $!\n";
die "framework version missing\n" unless defined $current;

# Update only current-release contexts. Compatibility boundaries, delivered-in versions,
# runtime versions, and historical design/implementation records must remain unchanged.
my $context = qr/
    Framework\ version:\s+
    | (?:solomonar|framework)\s+
    | "(?:cli|framework|project_framework|framework_version)":\s*"
    | In\ v
    | [Tt]he\ v(?=\Q$current\E\ catalog)
/x;

open my $paths, '-|', 'git', 'ls-files', '-z', '--',
    'assets', 'README.md', 'site/src', 'docs/cli/interface.md'
    or die "list release files: $!\n";
my @files = do { local $/ = "\0"; <$paths> };
close $paths or die "list release files failed\n";

for my $path (@files) {
    $path =~ s/\0\z//;
    next unless $path =~ /\.(?:md|njk)\z/;
    open my $input, '<', $path or die "read $path: $!\n";
    my $content = do { local $/; <$input> };
    close $input or die "close $path: $!\n";
    next unless $content =~ s/($context)\Q$current\E(?!\d|\.\d)/$1$next/g;
    open my $output, '>', $path or die "write $path: $!\n";
    print {$output} $content or die "write $path: $!\n";
    close $output or die "close $path: $!\n";
}
