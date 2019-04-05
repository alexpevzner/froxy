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

// ----- Initialization -----
//
// Page initialization
//
function init() {
    var rq = tproxy.GetServerParams();
    rq.OnSuccess = function (data) {
        tproxy.UiSetInput("addr", data.addr);
        tproxy.UiSetInput("login", data.login);
        tproxy.UiSetInput("password", data.password);
    };
}


init();

// vim:ts=8:sw=2:et
