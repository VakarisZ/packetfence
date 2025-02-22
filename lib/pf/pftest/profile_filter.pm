package pf::pftest::profile_filter;
=head1 NAME

pf::pftest::profile_filter

=head1 SYNOPSIS

 pftest profile_filter mac [name=value ...]

manipulate node entries

examples:

 pftest profile_filter 01:02:03:04:05:06 last_ssid=Bob


=head1 DESCRIPTION

pf::pftest::profile_filter

=cut

use strict;
use warnings;
use base qw(pf::cmd);
use pf::Connection::ProfileFactory;
use pf::util qw(clean_mac);
use pf::constants::exit_code qw($EXIT_SUCCESS);
use pf::constants;

sub parseArgs {
    my ($self) = @_;
    my ($mac, @args) = $self->args;
    $self->{mac} = clean_mac($mac);
    return $FALSE unless $self->{mac};
    return $self->_parse_attributes(@args);
}

sub _run {
    my ($self) = @_;
    my $profile = pf::Connection::ProfileFactory->instantiate($self->{mac}, $self->{params});
    my $name = $profile->name;
    print "Found '$name' profile for $self->{mac} \n";
    return $EXIT_SUCCESS;
}

=head2 _parse_attributes

parse and validate the arguments for 'pfcmd node add|edit' commands

=cut

sub _parse_attributes {
    my ($self,@attributes) = @_;
    my %params;
    for my $attribute (@attributes) {
        if($attribute =~ /^([a-zA-Z0-9_-]+)=(.*)$/ ) {
            $params{$1} = $2;
        } else {
            print STDERR "$attribute is badily formatted\n";
            return 0;
        }
    }
    $self->{params} = \%params;
    return 1;
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

