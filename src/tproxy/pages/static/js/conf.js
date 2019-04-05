//
// Configuration page script
//

"use strict";

// ----- Static variables -----
//
// Saved server parameters
//
var server_params = {};

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
    var password = document.getElementById("password");
    password.disabled = server_params.haskeys && tproxy.UiGetInput("usekey");
}

// ----- Initialization -----
//
// Page initialization
//
function init() {
    var rq = tproxy.GetServerParams();
    rq.OnSuccess = function (data) {
        server_params = data;

        tproxy.UiSetInput("addr", data.addr);
        tproxy.UiSetInput("login", data.login);
        tproxy.UiSetInput("password", data.password);
        tproxy.UiSetInput("usekey", data.usekey);

        var password = document.getElementById("password");
        var usekey = document.getElementById("usekey");
        var usekey_comment = document.getElementById("usekey.comment");

        password.disabled = data.haskeys && data.usekey;
        usekey.disabled = !data.haskeys;
        usekey_comment.hidden = data.haskeys;
    };
}


init();

// vim:ts=8:sw=2:et
