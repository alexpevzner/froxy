//
// Configuration page script
//

"use strict";

// ----- Server parameters -----
//
// Submit server parameters
//
function SubmitServerParams () {
    tproxy.SetServerParams(
        tproxy.UiGetInput("addr"),
        tproxy.UiGetInput("login"),
        tproxy.UiGetInput("password")
    );
}

// ----- SSH key management -----
//
// Generate SSH key
//
function GenKey() {
    alert("Not implemented");
}

//
// Copy public key to clipboard
//
function PubKeyCopy() {
    var el = document.getElementById("key-pubtext");
    var text = el.value;

    if (el.value) {
        el.select();
        document.execCommand("copy");
    }
}

//
// Save public key to file
//
function PubKeySave() {
    alert("Not implemented");
}

// ----- Initialization -----
//
// Page initialization
//
function init() {
    var rq = tproxy.GetServerParams();
    rq.OnSuccess = function (data) {
        tproxy._.debug("xxx", data);
        tproxy.UiSetInput("addr", data.addr);
        tproxy.UiSetInput("login", data.login);
        tproxy.UiSetInput("password", data.password);
    };
}


init();

// vim:ts=8:sw=2:et
