//
// Configuration page script
//

"use strict";

// ----- Static variables -----
//
// Saved server parameters
//
var server_params = {};
var haskeys = false;

// ----- Server parameters -----
//
// Submit server parameters
//
function SubmitServerParams () {
    tproxy.SetServerParams(
        tproxy.UiGetInput("addr"),
        tproxy.UiGetInput("login"),
        tproxy.UiGetInput("password"),
        tproxy.UiGetInput("usekey")
    );
}

//
// Called when "Use SSH keys" clicked
//
function UseKeysClicked () {
    UpdateConditionallyDisabled();
}

//
// Poll callback for server parameters
//
function PollServerParams (data) {
    server_params = data;

    tproxy.UiSetInput("addr", server_params.addr);
    tproxy.UiSetInput("login", server_params.login);
    tproxy.UiSetInput("password", server_params.password);
    tproxy.UiSetInput("usekey", server_params.usekey);

    UpdateConditionallyDisabled();
}

//
// Poll callbacks for keys
//
function PollKeys (data) {
    haskeys = data.length > 0;
    UpdateConditionallyDisabled();
}

//
// Update conditionally disabled fields
//
function UpdateConditionallyDisabled () {
    var password = document.getElementById("password");
    var usekey = document.getElementById("usekey");
    var usekey_comment = document.getElementById("usekey.comment");

    password.disabled = haskeys && tproxy.UiGetInput("usekey");
    usekey.disabled = !haskeys;
    usekey_comment.hidden = haskeys;
}


// ----- Initialization -----
//
// Page initialization
//
function init() {
    tproxy.BgPoll("/api/server", PollServerParams);
    tproxy.BgPoll("/api/keys", PollKeys);
}

init();

// vim:ts=8:sw=2:et
