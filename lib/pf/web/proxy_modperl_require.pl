#!/usr/bin/perl

=head1 NAME

captiveportal_modperl_require.pl - Pre-loading PacketFence's modules in Apache (mod_perl) for the Captive Portal

=cut

use strict;
use warnings;

BEGIN {
    use lib qw(/usr/local/pf/lib /usr/local/pf/lib_perl/lib/perl5);
    use pf::log 'service' => 'httpd.proxy', reinit => 1;
}

use pf::web::interceptproxy;


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
