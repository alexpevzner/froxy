# Froxy - HTTP over SSH proxy

## Introduction

Froxy is the HTTP over SSH proxy

It allows you to visit web sites as via VPN, but using your own SSH server,
hosted somewhere in the Internet.

The only things you need is the Froxy itself and some Linux server accessible
via SSH. No special software needs to be installed or configured on the server
side, just SSH server that you already have.

At the server-side almost no configuration is required: just append your SSH
public key to the `.ssh/authorized_keys`. Password-based authentication is also
possible, though not recommended.

At the client side, just install Froxy and add it as HTTP proxy to your web
browser configuration.

Froxy is friendly program. You won't need to edit any cryptic configuration
files. All configuration is web-based and can be done in your browser. Just
click Froxy icon on a desktop, and it will open your Froxy configuration page
in your favorite web browser.

Unlike many VPNs, Froxy only relays explicitly configured sites via the
server, connections to all other sites go directly from your
local computer.

HTTP support in Froxy is fairly complete and using it doesn't imply any
limitations to the normal web braising. Even FTP URLs are supported (though FTP
support is abandoned in most modern browsers).

Froxy can be build for and works on Linux and Windows. Volunteer wanting
to port to any other platform are welcome!
