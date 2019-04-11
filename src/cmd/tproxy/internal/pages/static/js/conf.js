//
// Configuration page script
//

"use strict";

// ----- Static variables -----
//
// Saved parameters
//
var saved_server_params = {};
var saved_keys = [];

// ----- Authentication method selection -----
//
// Update auth method selection control
//
// Called when either keys or server parameters changed
//
function AuthMethodUpdate() {
    var keyid = saved_server_params.keyid || "";
    var keyid_ok = false;
    var auth = document.getElementById("auth");
    var method = auth.value;
    var method_is_valid_keyid = false;
    var i, elm, s, key;

    // Purge selection options.
    while (auth.children.length > 2) {
        auth.children[auth.children.length-1].remove();
    }

    // Rebuild selection options
    for (i = 0; i < saved_keys.length; i ++) {
        key = saved_keys[i];

        elm = document.createElement("option");
        elm.value = key.id;

        s = "Key " + (i + 1) + " (";
        s += key.type;
        if (key.comment) {
            s += ", " + key.comment;
        }
        s += ")";
        elm.innerText = s;

        auth.appendChild(elm);

        if (keyid == key.id) {
            keyid_ok = true;
        }

        if (method == key.id) {
            method_is_valid_keyid = true;
        }
    }

    // Restore previous selection, if possible
    if (method_is_valid_keyid) {
        // Do nothing
    } else if (keyid_ok) {
        method = keyid;
    } else if (saved_server_params.password) {
        method = "auth.password";
    } else {
        method = "auth.none";
    }

    auth.value = method;

    // Enable/disable password input
    PasswordConditionallyEnable();
}

//
// Authentication method onchange callback
//
function AuthMethodOnChange () {
    PasswordConditionallyEnable();
}

// ----- Password -----
//
// Enable/Disable password
//
function PasswordConditionallyEnable () {
    var password = document.getElementById("password");
    var auth = tproxy.UiGetInput("auth");

    password.disabled = auth != "auth.password";
}

// ----- User inputs callbacks -----
//
// Submit server parameters
//
function SubmitServerParams () {
    var keyid = tproxy.UiGetInput("auth");

    switch (keyid) {
    case "auth.none":
    case "auth.password":
        keyid = "";
    }

    tproxy.SetServerParams(
        tproxy.UiGetInput("addr"),
        tproxy.UiGetInput("login"),
        tproxy.UiGetInput("password"),
        keyid
    );
}

// ----- Poll callbacks -----
//
// Poll callback for server parameters
//
function PollServerParams (data) {
    saved_server_params = data;

    tproxy.UiSetInput("addr", saved_server_params.addr);
    tproxy.UiSetInput("login", saved_server_params.login);
    tproxy.UiSetInput("password", saved_server_params.password);

    AuthMethodUpdate();
}

//
// Poll callbacks for keys
//
function PollKeys (data) {
    saved_keys = data;
    AuthMethodUpdate();
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
