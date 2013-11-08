#!/usr/bin/env perl
use strict;
use warnings;
use Test::More;
use FindBin qw/$Bin/;
use File::Temp qw/tempdir/;

my $dir = tempdir(CLEANUP=>1);

my $testdir = "$Bin/";
my $linefan = "$Bin/../linefan -q";

my $cmd = 'date';
system("cd $dir; $linefan $cmd");
ok(-d "$dir/.linefan", "creates .linefan");
ok(-f "$dir/.linefan/$cmd", "'linefan $cmd' creates .linefan/$cmd");

$cmd = 'ls -la|wc -l';
system("cd $dir; $linefan '$cmd'");
my $expected = "ls -la|wc -l";
ok(-f "$dir/.linefan/$expected", "'linefan $cmd' creates .linefan/$expected");

done_testing();



