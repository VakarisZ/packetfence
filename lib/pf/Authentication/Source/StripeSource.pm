package pf::Authentication::Source::StripeSource;
=head1 NAME

pf::Authentication::Source::StripeSource

=cut

=head1 DESCRIPTION

pf::Authentication::Source::StripeSource

=cut

use strict;
use warnings;
use Moose;
use HTTP::Status qw(is_success);
use WWW::Curl::Easy;
use pf::Curl;
use JSON::MaybeXS;
use URI::Escape::XS qw(uri_escape);
use List::Util qw(pairmap);
use Digest::SHA qw(hmac_sha1_hex);

use pf::config qw($default_pid);
use pf::constants qw($FALSE $TRUE);
use pf::Authentication::constants;
use pf::util;
use pf::log;
use pf::error qw(is_error);

extends 'pf::Authentication::Source::BillingSource';
with 'pf::Authentication::CreateLocalAccountRole';
our $logger = get_logger;

=head2 Attributes

=head2 class

=cut

has '+class' => (default => 'billing');

has '+type' => (default => 'Stripe');

has 'host' => (is => 'rw', default => 'api.stripe.com');

has 'proto' => (is => 'rw', default => 'https');

has 'port' => (is => 'rw', default => 443);

has 'publishable_key' => (is => 'rw', required => 1);

has 'secret_key' => (is => 'rw', required => 1);

has 'style' => (is => 'rw', default => 'charge');

has 'domains' => (is => 'rw', default => '*.stripe.com,stripe.network,*.stripe.network');

has 'customer_portal' => (is => 'rw', default => 'disabled');

=head2 url

  The url to the rpc message to

=cut

sub base_url {
    my ($self) = @_;
    my $proto  = $self->proto;
    my $host   = $self->host;
    my $port   = $self->port;
    return "${proto}://${host}:${port}";
}

=head2 curl

  Creates a curl object to connect to the rpc server

=cut

sub curl {
    my ($self, $function) = @_;
    my $curl = pf::Curl::easy();
    $curl->setopt(CURLOPT_HEADER,               0);
    $curl->setopt(CURLOPT_DNS_USE_GLOBAL_CACHE, 0);
    $curl->setopt(CURLOPT_NOSIGNAL,             1);
    $curl->setopt(CURLOPT_HTTPHEADER, ['Accept: application/json', 'Accept-Language: en_US']);
    if ($self->proto eq 'https') {

        # Removed SSL verification
        $curl->setopt(CURLOPT_SSL_VERIFYHOST, 0);
        $curl->setopt(CURLOPT_SSL_VERIFYPEER, 0);
        $curl->setopt(CURLOPT_USERNAME, $self->secret_key);
        $curl->setopt(CURLOPT_PASSWORD, '');
    }
    return $curl;
}

=head2 _send_json

send json data

=cut

sub _send_json {
    my ($self, $curl, $path, $object) = @_;
    $curl->setopt(CURLOPT_HTTPHEADER, ["Content-Type: application/json"]);
    $self->_set_url($curl, $path);
    my $data = encode_json $object;
    $self->_set_body($curl, $data);
    return $self->_do_request($curl);
}

=head2 _get_json

GET JSON data from the Stripe API

=cut

sub _get_json {
    my ($self, $curl, $path) = @_;
    $self->_set_url($curl, $path);
    return $self->_do_request($curl);
}

=head2 _send_form

send form data

=cut

sub _send_form {
    my ($self, $curl, $path, $object) = @_;
    $self->_set_url($curl, $path);
    my $data = encode_form($object);
    $self->_set_body($curl, $data);
    return $self->_do_request($curl);
}

sub encode_form {
    my ($object) = @_;
    my @values;
    while( my ($key, $val) = each %$object) {
        push @values, (uri_escape($key) . "=" . uri_escape($val));
    }
    return join ('&',@values);
}

=head2 _set_body

Set the body for the curl object

=cut

sub _set_body {
    my ($self, $curl, $data) = @_;
    $curl->setopt(CURLOPT_POSTFIELDSIZE, length($data));
    $curl->setopt(CURLOPT_POSTFIELDS,    $data);
}

=head2 _set_url

Set the url for the curl object

=cut

sub _set_url {
    my ($self, $curl, $path) = @_;
    my $base_url = $self->base_url;
    $path =~ s#^\/##;
    my $url = "$base_url/$path";
    $curl->setopt(CURLOPT_URL, $url);
}

=head2 _do_request

Send the request and return the status code and body

=cut

sub _do_request {
    my ($self, $curl) = @_;
    my $response_body;
    $curl->setopt(CURLOPT_WRITEDATA, \$response_body);
    my $curl_return_code = $curl->perform;
    if ($curl_return_code == 0) {
        my $response_code = $curl->getinfo(CURLINFO_HTTP_CODE);
        my $response      = decode_json($response_body);
        return ($response_code, $response);
    }
    else {
        die "Error send request: " . $curl->errbuf;
    }
}

=head2 prepare_payment

=cut

sub prepare_payment {
    my ($self, $session, $tier, $params, $uri) = @_;
    return {};
}


=head2 charge

=cut

sub charge {
    my ($self, $tier, $token) = @_;
    my $object = {
        amount   => int($tier->{price} * 100),
        currency => $self->currency,
        source   => $token,
        description => $tier->{description},
    };
    my ($code,$data) = $self->_send_form($self->curl, "v1/charges", $object);
}

sub verify {
    my ($self, $session, $params, $uri) = @_;
    use Data::Dumper; get_logger->debug(Dumper($params));
    my $token = $params->{stripeToken};
    die "No Token found" unless defined $token;
    my $style = $self->style;
    my ($code, $response);
    if ($style eq 'charge') {
        ($code, $response) = $self->charge($session->{tier}, $token);
    } elsif ($style eq 'subscription') {
        ($code, $response) = $self->subscribe_customer($session, $session->{tier}, $token);
    }
    unless(is_success($code)) {
        die "Unable to process payment : " . $response->{error}{message};
    }
    return {};
}

sub subscribe_customer {
    my ($self, $session, $tier, $token) = @_;

    if(ref($token) eq "ARRAY") {
        return (500, {error => {message => "Invalid request, please try again."}});
    }

    my $object = {
        plan   => $tier->{id},
        source => $token,
        email => $session->{email},
        description => $tier->{description},
        'metadata[mac_address]' => $session->{billed_mac},
    };

    if($session->{request_fields}->{firstname} && $session->{request_fields}->{lastname}) {
        $object->{name} = $session->{request_fields}->{firstname} . " " . $session->{request_fields}->{lastname};
    }
    if($session->{request_fields}->{address}) {
        $object->{'address[line1]'} = $session->{request_fields}->{address};
    }

    my ($code, $data);
    if(my $cus = $self->get_customer_by_email($session->{email})) {
        use Data::Dumper ; use pf::log ; get_logger->info(Dumper($cus));
        if($cus->{subscriptions} && $cus->{subscriptions}->{data}->[0]) {
            $self->cancel_subscription($cus->{subscriptions}->{data}->[0]->{id});
        }
        ($code, $data) = $self->_send_form($self->curl, "v1/customers/".$cus->{id}, $object);
    }
    else {
        ($code, $data) = $self->_send_form($self->curl, "v1/customers", $object);
    }

    my $subscriptions = $self->_get_json($self->curl, "v1/subscriptions?customer=".uri_escape($data->{id}));
    if($subscriptions->{data}->[0]->{status} ne "active") {
        my $msg = "The subscription is not active after payment(".$subscriptions->{data}->[0]->{status}."). Card payment didn't go through.";
        $logger->error($msg);
        return (500, {error => {message => $msg}});
    }

    $session->{stripe_customer_id} = $data->{id};
    return ($code, $data);
}

sub handle_hook {
    my ($self, $headers, $content) = @_;
    my $object = decode_json $content;
    my $type   = $object->{type};
    $type =~ s/\./_/g;
    my $handler = "handle_$type";
    if ($self->can($handler)) {
        my $status;
        eval {
            $status = $self->$handler($headers, $object);
        };
        if ($@) {
            $logger->error($@);
            return 500;
        }
        return $status;
    }
    $logger->warn("Unsupported type $type recieved");
    return $self->handle_event($headers, $object);
}

sub handle_event {
    return 200;
}

sub handle_customer_subscription_deleted {
    my ($self, $headers, $object) = @_;
    my $customer_id = $object->{data}{object}{customer};
    my ($status, $customer) = $self->get_customer($customer_id);
    my $email = $customer->{email};
    my $client_mac = $customer->{metadata}{mac_address};
    get_logger->info("Handling subscription deletion for customer $customer->{id}");
    # Can't import at the top as this cannot use pf::config
    require pf::node;
    pf::node::node_deregister($client_mac);
    $self->send_mail_for_event(
        $object,
        email   => $customer->{email},
        subject => "Your Subscription has been canceled",
        mac => $client_mac,
    );
    return 200;
}

sub send_mail_for_event {
    my ($self, $event, %data) = @_;
    if($self->can_send_mail_for_event($event)) {
        my $type = $event->{type};
        $data{event} = $event;
        require pf::config::util;
        pf::config::util::send_email("billing_stripe_$type", $data{'email'}, $data{'subject'}, \%data);
    }
}

sub get_customer_by_email {
    my ($self, $email) = @_;
    return $self->get_customers_by_email($email)->[0];
}

sub get_customers_by_email {
    my ($self, $email) = @_;
    my $curl = $self->curl;
    my $path = "/v1/customers?email=".uri_escape($email)."&expand[]=data.subscriptions";
    $self->_set_url($curl, $path);
    my $res = $self->_do_request($curl);
    return $res->{data};
}

sub get_customer {
    my ($self, $customer) = @_;
    my $curl = $self->curl;
    my $path = "/v1/customers/$customer?expand[]=subscriptions";
    $self->_set_url($curl, $path);
    return $self->_do_request($curl);
}

sub cancel_subscription {
    my ($self, $subscription_id) = @_;
    my $curl = $self->curl;
    my $path = "/v1/subscriptions/$subscription_id?prorate=true&invoice_now=true";
    $self->_set_url($curl, $path);
    $curl->setopt(CURLOPT_CUSTOMREQUEST, "DELETE");
    return $self->_do_request($curl);
}

sub can_send_mail_for_event {
    my ($self, $event) = @_;
    return 1;
}

=head2 mandatoryFields

List of mandatory fields for this source

=cut

sub mandatoryFields {
    return qw(email);
}

sub handleCancelLink {
    my ($self, $subscription_id, $query_args) = @_;
    my $validator = $query_args->{validator};
    my $wants_validator = hmac_sha1_hex($subscription_id.$self->id.$self->publishable_key, $self->secret_key);
    if($validator eq $wants_validator) {
        my ($status, $subscription) = $self->cancel_subscription($subscription_id); 
        if(is_error($status) || !$subscription) {
            if($status == 404) {
                return ($FALSE, "This subscription has already been canceled.");
            }
            else {
                return ($FALSE, "Unable to cancel your subscription. Please contact your local support staff.");
            }
        }
        else {
            get_logger->info("Canceled subscription $subscription_id");
            return ($TRUE, "Canceled subscription");
        }
    }
    else {
        my $msg = "Validator of the request doesn't match the subscription ID.";
        get_logger->error($msg);
        return ($FALSE, $msg);
    }
}

sub cancelLinkPath {
    my ($self, $parameters, $tier, $session) = @_;
    return "status/billing/cancel_subscription/".$self->id."/".$session->{subscription_id} . "?validator=" . hmac_sha1_hex($session->{subscription_id}.$self->id.$self->publishable_key, $self->secret_key);;
}

sub additionalConfirmationInfo {
    my ($self, $parameters, $tier, $session) = @_;
    if($session->{stripe_customer_id}) {
        my ($status, $customer) = $self->get_customer($session->{stripe_customer_id});
        if(is_error($status)) {
            get_logger->error("Unable to fetch stripe customer: ".Dumper($customer));
            return ();
        }
        $session->{subscription_id} = $customer->{subscriptions}->{data}->[0]->{id};
        return (cancel_link => $self->cancelLinkPath($parameters, $tier, $session));
    }
    else {
        return ();
    }
}

sub setupStripeCustomerPortal {
    my ($self, $cus_id, $return_url) = @_;
    my ($code, $data) = $self->_send_form($self->curl, "v1/billing_portal/sessions", {customer => $cus_id, return_url => $return_url});
    if(is_success($code)) {
        return $data->{url};
    } else {
        use Data::Dumper ; get_logger->error("Failed to setup Stripe Customer Portal session. Code $code. Response: ", Dumper($data));
        return undef;       
    }
}

=head1 AUTHOR

Inverse inc. <info@inverse.ca>

=head1 COPYRIGHT

Copyright (C) 2005-2023 Inverse inc.

=head1 LICENSE

This program is free software; you can redistribute it and::or
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
