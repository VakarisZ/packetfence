package pfconfig::namespaces::config::EventLoggers;

=head1 NAME

pfconfig::namespaces::config::template

=cut

=head1 DESCRIPTION

pfconfig::namespaces::config::template

This module creates the configuration hash associated to somefile.conf

=cut

use strict;
use warnings;

use pfconfig::namespaces::config;
use pf::file_paths qw($event_loggers_config_file);
use pf::constants::eventLogger;

use base 'pfconfig::namespaces::config';

sub init {
    my ($self) = @_;
    $self->{file} = $event_loggers_config_file;
}

sub build_child {
    my ($self) = @_;
    my %tmp = %{ $self->{cfg} };
    my %namespaces;
    while ( my ( $k, $v ) = each %tmp ) {
        $v->{id} = $k;
        my $ns = $v->{namespaces};
        my @names = defined $ns && $ns ne '' ? split( /\s*,\s*/, $ns ) : @pf::constants::eventLogger::Namespaces;
        for my $n (@names) {
            push @{$namespaces{$n}}, {%$v} ;
        }
    }

    return \%namespaces;
}


=head1 AUTHOR

Inverse inc. <info@inverse.ca>

=head1 COPYRIGHT

Copyright (C) 2005-2023 Inverse inc.

=head1 LICENSE

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
as published by the Free Software Foundation; either version 2
of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program; if not, write to the Free Software
Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301,
USA.

=cut

1;

# vim: set shiftwidth=4:
# vim: set expandtab:
# vim: set backspace=indent,eol,start:

