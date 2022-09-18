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
    var method_ok = method == "auth.password";
    var i, elm, s, key;

    // Purge selection options.
    while (auth.children.length > 2) {
        auth.removeChild(auth.children[auth.children.length-1]);
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
            method_ok = true;
        }
    }

    // Restore previous selection, if possible
    if (method_ok) {
        // Do nothing
    } else if (keyid_ok) {
        method = keyid;
    } else if (!keyid && saved_server_params.password) {
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
    var auth = froxy.UiGetInput("auth");

    password.disabled = auth != "auth.password";
}

// ----- User inputs callbacks -----
//
// Submit server parameters
//
function SubmitServerParams () {
    var keyid = froxy.UiGetInput("auth");

    switch (keyid) {
    case "auth.none":
    case "auth.password":
        keyid = "";
    }

    froxy.SetServerParams(
        froxy.UiGetInput("addr"),
        froxy.UiGetInput("login"),
        froxy.UiGetInput("password"),
        keyid
    );
}

// ----- Poll callbacks -----
//
// Poll callback for server parameters
//
function PollServerParams (data) {
    saved_server_params = data;

    froxy.UiSetInput("addr", saved_server_params.addr);
    froxy.UiSetInput("login", saved_server_params.login);
    froxy.UiSetInput("password", saved_server_params.password);

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
    froxy.BgPoll("/api/server", PollServerParams);
    froxy.BgPoll("/api/keys", PollKeys);
}

window.onload = init;

// vim:ts=8:sw=2:et
