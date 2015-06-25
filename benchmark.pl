#!/usr/bin/env perl
use strict;
use warnings;
use utf8;
use feature "say";

use Test::TCP;
use Proc::Guard;
use Data::Dumper;
use Parallel::Benchmark;
use Cache::Memcached::Fast;

my $katsubushi_port = Test::TCP::empty_port();
my $katsubushi = Proc::Guard->new(
    command => [
        "go", "run", "cmd/katsubushi/main.go",
        "-port=${katsubushi_port}",
        "-worker-id=1",
    ],
);
Test::TCP::wait_port($katsubushi_port);

my $client = Cache::Memcached::Fast->new({ servers => ["localhost:${katsubushi_port}"] });

sub bench {
    my ($concurrency) = @_;

    my $pb = Parallel::Benchmark->new(
        time => 10,
        concurrency => $concurrency,
        benchmark => sub {
            $client->get("id");
            return 1;
        },
    );

    $pb->run;
}

say "serial";
bench(1);

say "parallel";
bench(4);

__END__

$ perl benchmark.pl
INFO[0000] Listening at [::]:50585
serial
2015-04-21T15:27:11 [21263] [INFO] starting benchmark: concurrency: 1, time: 10
2015-04-21T15:27:22 [21263] [INFO] done benchmark: score 97526, elapsed 10.012 sec = 9741.033 / sec
parallel
2015-04-21T15:27:22 [21263] [INFO] starting benchmark: concurrency: 4, time: 10
2015-04-21T15:27:33 [21263] [INFO] done benchmark: score 219615, elapsed 10.021 sec = 21914.999 / sec
perl benchmark.pl  4.19s user 11.40s system 63% cpu 24.393 total
